package coreSystem

import (
	"math"
	"math/rand"
	"time"
)

// GreenhouseSimulator models the thermal and humidity dynamics of a greenhouse
type GreenhouseSimulator struct {
	state         GreenhouseState
	lastState     GreenhouseState
	lastTime      time.Time
	heaterDelay   time.Duration
	delayedHeater *DelayBuffer
	startTime     time.Time // Track start time for dynamic environmental changes

	// Physical parameters
	thermalMass   float64 // Heat capacity
	heatLossCoeff float64 // Heat loss rate to ambient // TODO: is this nonconstant?
	humidityTau   float64 // Humidity response time constant (seconds)  // TODO: is this nonconstant?

	// Noise simulation
	tempNoise     float64 // Standard deviation of temperature sensor noise (F)
	humidityNoise float64 // Standard deviation of humidity sensor noise (%)

	// External condition dynamics (day/night cycle simulation)
	baseExternalTemp          float64 // Average external temperature (F)
	externalTempAmplitude     float64 // Temperature variation (F) - peak-to-trough
	baseExternalHumidity      float64 // Average external humidity (%)
	externalHumidityAmplitude float64 // Humidity variation (%) - peak-to-trough
	cycleLength               float64 // Duration of full day/night cycle (seconds)

	// Other
	rhClamper FloatBound

	rnd *rand.Rand
}

// NewGreenhouseSimulator creates a new greenhouse simulator
func NewGreenhouseSimulator(
	initialTemp, externalTemp, initialHumidity, externalHumidity float64,
	heaterDelay time.Duration,
	rnd *rand.Rand,
) *GreenhouseSimulator {
	now := time.Now()
	if rnd == nil {
		tempSeed := now.UnixNano() // TODO: ok?
		rnd = rand.New(rand.NewSource(tempSeed))
	}

	// Calculate delay buffer size: number of steps to store
	// Assume ~60 calls per minute, so 3 Min Max = 180 steps
	delaySteps := int(math.Ceil(heaterDelay.Seconds() * 2)) // 2 Hz sampling rate
	tempState := GreenhouseState{
		Time: now,
		Temp: InternalExternal{
			Internal: initialTemp,
			External: externalTemp,
		},
		Humidity: InternalExternal{
			Internal: initialHumidity,
			External: externalHumidity,
		},
		HeaterPowerOutput: 0.0,
		FoggerPowerOutput: 0.0,
		FanPowerOutput:    0.0,
	}
	return &GreenhouseSimulator{
		state:     tempState,
		lastState: tempState,
		lastTime:  now,
		startTime: now,
		//heaterDelay:   heaterDelay,
		delayedHeater: NewDelayBuffer(delaySteps),
		thermalMass:   100.0, // Heat capacity (J/K) // TODO: ???
		heatLossCoeff: 0.5,   // Heat loss coefficient // TODO: ???
		humidityTau:   30.0,  // Humidity time constant (seconds) // TODO: ???
		tempNoise:     0.3,   // +/- 0.3 F noise // TODO: ???
		humidityNoise: 1.5,   // +/- 1.5% RH noise // TODO: ???
		// Dynamic external conditions (day/night cycle)
		baseExternalTemp:          externalTemp,         // Average external temperature
		externalTempAmplitude:     10.0,                 // ±10°F swing (day/night) // TODO: ???
		baseExternalHumidity:      externalHumidity,     // Average external humidity
		externalHumidityAmplitude: 20.0,                 // ±20% RH swing (day/night) // TODO: ???
		cycleLength:               3600.0,               // 1-hour cycle (for testing; 24h = 86400s in real world) // TODO: ???
		rhClamper:                 newBound(20.0, 99.0), // TODO: ok?
		rnd:                       rnd,
	}
}

// Step advances the simulation by one control step
func (g *GreenhouseSimulator) Step(heaterPower float64) GreenhouseState {
	now := time.Now()
	dt := now.Sub(g.lastTime).Seconds()
	if dt <= 0 {
		dt = 0.1 // Minimum time step // TODO: ok?
	}

	// Calculate time-varying external conditions (day/night cycle)
	elapsedSeconds := now.Sub(g.startTime).Seconds()
	phase := (elapsedSeconds / g.cycleLength) * 2.0 * math.Pi // Convert to radians

	// Sinusoidal variation: base ± amplitude
	externalTempVariation := g.externalTempAmplitude * 0.5 * math.Sin(phase)
	externalHumidityVariation := g.externalHumidityAmplitude * 0.5 * math.Sin(phase+math.Pi) // 180° out of phase

	g.state.Temp.External = g.baseExternalTemp + externalTempVariation
	g.state.Humidity.External = math.Max(0, math.Min(100, g.baseExternalHumidity+externalHumidityVariation))

	// Store current heater command in delay buffer + Get the delayed heater output (from N steps ago)
	delayedHeaterPower := g.delayedHeater.Push(heaterPower)

	// ========== Temperature Dynamics ==========
	// dT/dt = (heater_input - heat_loss) / thermal_mass
	// heater_input = heater_power * max_heating_rate (scaled 0-1)
	// heat_loss = loss_coefficient * (T_internal - T_external)

	maxHeatingRateTempDegPerMin := 1.1 // Degrees // TODO: ok?
	secondsPerMinute := 60.0
	maxHeatingRatePerSec := maxHeatingRateTempDegPerMin / secondsPerMinute // 1.1 degrees per minute
	heatInput := delayedHeaterPower * maxHeatingRatePerSec
	heatLoss := g.heatLossCoeff * (g.state.Temp.Internal - g.state.Temp.External) / g.thermalMass

	// Add some stochasticity to thermal response
	jitter := (g.rnd.Float64() - 0.5) * 0.1 // +/- 5% random variation
	tempDelta := (heatInput - heatLoss) * dt * (1.0 + jitter)
	g.state.Temp.Internal += tempDelta

	// ========== Humidity Dynamics ==========
	// Simplified exponential approach toward equilibrium
	// Heating reduces relative humidity, water addition increases it

	// Target humidity depends on heater (evaporative cooling) and fogger (water addition)
	// For now, model humidity drift toward external + heating effect
	heatingFactor := -0.3 * delayedHeaterPower // Heating dries the air
	externalInfluence := 0.02 * (g.state.Humidity.External - g.state.Humidity.Internal)

	humidityRate := (externalInfluence + heatingFactor) / g.humidityTau
	jitterHum := (g.rnd.Float64() - 0.5) * 0.05
	humidityDelta := humidityRate * dt * (1.0 + jitterHum)
	g.state.Humidity.Internal += humidityDelta

	// Clamp humidity to realistic ranges
	g.state.Humidity.Internal, _ = g.rhClamper.Clamp(g.state.Humidity.Internal) // TODO: _ ok?

	// Add sensor noise
	tempNoise := g.rnd.NormFloat64() * g.tempNoise
	humidityNoise := g.rnd.NormFloat64() * g.humidityNoise

	g.state.Time = now
	g.state.HeaterPowerOutput = heaterPower

	// Store noisy values for output
	noisyTemp := g.state.Temp.Internal + tempNoise
	noisyHumidity := g.state.Humidity.Internal + humidityNoise

	// Clamp noisy humidity to valid range
	if noisyHumidity < 0.0 {
		noisyHumidity = 0.0
	}
	if noisyHumidity > 100.0 {
		noisyHumidity = 100.0
	}

	result := g.state
	result.Temp.Internal = noisyTemp
	result.Humidity.Internal = noisyHumidity

	g.lastState = g.state
	g.lastTime = now

	return result
}

// Current returns the current greenhouse state without advancing time
func (g *GreenhouseSimulator) Current() GreenhouseState {
	return g.state
}

// GetState returns current state with sensor noise applied
func (g *GreenhouseSimulator) GetState() GreenhouseState {
	tempNoise := g.rnd.NormFloat64() * g.tempNoise
	humidityNoise := g.rnd.NormFloat64() * g.humidityNoise

	s := g.state
	s.Temp.Internal += tempNoise
	s.Humidity.Internal = math.Max(0, math.Min(100, s.Humidity.Internal+humidityNoise))
	return s
}

// NewDelayBuffer creates a circular buffer for time-delayed signals
func NewDelayBuffer(size int) *DelayBuffer {
	return &DelayBuffer{
		buffer: make([]float64, size),
		index:  0,
		size:   size,
		filled: false,
	}
}

// Push adds a new value and returns what would be read // TODO: does not return?
func (db *DelayBuffer) Push(value float64) float64 {
	db.buffer[db.index] = value
	db.index = (db.index + 1) % db.size
	if db.index == 0 {
		db.filled = true
	}
	return db.Get()
}

// Get returns the value that was pushed N steps ago
func (db *DelayBuffer) Get() float64 {
	if !db.filled && db.index == 0 {
		return 0.0
	}
	return db.buffer[db.index]
}

// ReadSensors implements GreenhouseEnvironment interface
// Returns the current greenhouse state as a sensor reading
func (g *GreenhouseSimulator) ReadSensors() (GreenhouseState, error) {
	return g.GetState(), nil
}

// SendControl implements GreenhouseEnvironment interface
// Accepts a heater power command and advances the simulation
func (g *GreenhouseSimulator) SendControl(output ControlOutput) error {
	g.Step(output.HeaterPower)
	return nil
}

// GetCurrentHeaterPower implements GreenhouseEnvironment interface
// Returns the current heater power output
func (g *GreenhouseSimulator) GetCurrentHeaterPower() float64 {
	return g.GetState().HeaterPowerOutput
}

// Close implements GreenhouseEnvironment interface
// Cleanup for simulator (no-op, but needed for interface compliance)
func (g *GreenhouseSimulator) Close() error {
	return nil
}
