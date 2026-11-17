package coreSystem

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// RealGreenhouseEnvironment represents an actual physical greenhouse with real sensors and actuators
// This is the production implementation that replaces the simulator
type RealGreenhouseEnvironment struct {
	// Sensor data
	sensorMutex       sync.RWMutex
	lastSensorReading GreenhouseState
	sensorReadError   error

	// Actuator state
	actuatorMutex     sync.RWMutex
	lastHeaterPower   float64 // TODO: OTHER ACTUATORS
	lastActuatorError error
	actuatorFailsafe  bool // Safety mode when actuator fails

	// Configuration
	sensorUpdateInterval   time.Duration
	actuatorUpdateInterval time.Duration

	// Hardware interfaces (to be set by user)
	hardwareSensors   HardwareSensors
	hardwareActuators HardwareActuators

	// Monitoring
	readCount       int64
	writeCount      int64
	lastReadTime    time.Time
	lastWriteTime   time.Time
	communicationOK bool // TODO: if communication is not ok, what do we want to do?
}

// HardwareSensors interface for actual sensor hardware
type HardwareSensors interface {
	// ReadTemperatureHumidity reads temperature and humidity from sensors
	// Returns: internalTemp(°F), internalHumidity(%), externalTemp(°F), externalHumidity(%), error
	ReadTemperatureHumidity() (float64, float64, float64, float64, error)

	// Initialize sensor hardware
	Initialize() error

	// Cleanup sensor resources
	Close() error
}

// HardwareActuators interface for actual actuator hardware
type HardwareActuators interface {
	// SetHeaterPower sets the heater power level (0.0 = off, 1.0 = full) // TODO: does this also set the historical heater power?
	SetHeaterPower(power float64) error

	// GetHeaterStatus returns current heater state
	GetHeaterStatus() (float64, error)

	// Initialize actuator hardware
	Initialize() error

	// Cleanup actuator resources
	Close() error

	// EmergencyShutdown stops all actuators immediately
	EmergencyShutdown() error
}

// NewRealGreenhouseEnvironment creates a real greenhouse environment with actual hardware
func NewRealGreenhouseEnvironment(
	sensors HardwareSensors,
	actuators HardwareActuators,
	sensorUpdateInterval time.Duration,
	actuatorUpdateInterval time.Duration,
) (*RealGreenhouseEnvironment, error) {
	if sensors == nil || actuators == nil {
		return nil, fmt.Errorf("sensors and actuators must not be nil")
	}

	// Initialize hardware
	if err := sensors.Initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize sensors: %w", err)
	}

	if err := actuators.Initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize actuators: %w", err)
	}

	env := &RealGreenhouseEnvironment{
		lastSensorReading: GreenhouseState{
			Time: time.Now(),
		},
		sensorUpdateInterval:   sensorUpdateInterval,
		actuatorUpdateInterval: actuatorUpdateInterval,
		hardwareSensors:        sensors,
		hardwareActuators:      actuators,
		communicationOK:        true,
		lastHeaterPower:        0.0,
		actuatorFailsafe:       false,
	}

	// Start background sensor reader
	go env.backgroundSensorReader()

	return env, nil
}

// backgroundSensorReader continuously reads sensor data in the background
func (r *RealGreenhouseEnvironment) backgroundSensorReader() {
	ticker := time.NewTicker(r.sensorUpdateInterval)
	defer ticker.Stop()

	for range ticker.C {
		internalTemp, internalRH, externalTemp, externalRH, err := r.hardwareSensors.ReadTemperatureHumidity()

		r.sensorMutex.Lock()
		r.lastSensorReading = GreenhouseState{
			Temp: InternalExternal{
				Internal: internalTemp,
				External: externalTemp,
			},
			Humidity: InternalExternal{
				Internal: internalRH,
				External: externalRH,
			},
			Time: time.Now(),
		}
		r.sensorReadError = err
		r.lastReadTime = time.Now()
		r.readCount++

		if err != nil {
			r.communicationOK = false
			log.Printf("[GREENHOUSE] Sensor read error: %v", err)
		} else {
			r.communicationOK = true
		}
		r.sensorMutex.Unlock()
	}
}

// ReadSensors implements SensorReader interface - reads latest sensor data
func (r *RealGreenhouseEnvironment) ReadSensors() (GreenhouseState, error) {
	r.sensorMutex.RLock()
	defer r.sensorMutex.RUnlock()

	// Check if sensor data is stale
	if time.Since(r.lastSensorReading.Time) > r.sensorUpdateInterval*3 {
		return GreenhouseState{}, fmt.Errorf("sensor data is stale (age: %v)", time.Since(r.lastSensorReading.Time))
	}

	if r.sensorReadError != nil {
		return GreenhouseState{}, r.sensorReadError
	}

	return r.lastSensorReading, nil
}

type FloatBound struct {
	Min, Max float64
}

func newBound(min, max float64) FloatBound {
	return FloatBound{
		Min: min,
		Max: max,
	}
}
func (fc FloatBound) Clamp(val float64) (clampedValue float64, changed bool) {
	if val < fc.Min {
		return fc.Min, true
	}
	if val > fc.Max {
		return fc.Max, true
	}
	return val, false
}
func (fc FloatBound) IsInBound(val float64) bool {
	return !fc.OutOfBounds(val)
}
func (fc FloatBound) OutOfBounds(val float64) bool {
	return val <= fc.Min || val >= fc.Max
}
func (fc FloatBound) DeltaMinMax() float64 {
	return fc.Max - fc.Min
}

// SendControl implements GreenhouseEnvironment interface - sends control commands
func (r *RealGreenhouseEnvironment) SendControl(output ControlOutput) error {
	// Clamp heater power to valid range
	heaterPower, _ := percentageClamper.Clamp(output.HeaterPower) // TODO: _ ok?

	// Send to hardware
	err := r.hardwareActuators.SetHeaterPower(heaterPower)

	r.actuatorMutex.Lock()
	r.lastHeaterPower = heaterPower
	r.lastActuatorError = err
	r.lastWriteTime = time.Now()
	r.writeCount++

	if err != nil {
		r.communicationOK = false
		log.Printf("[GREENHOUSE] Actuator write error: %v", err)
		// Failsafe: turn off heater on error
		r.actuatorFailsafe = true
		_ = r.hardwareActuators.SetHeaterPower(0.0)
		// TODO: set last heater power????
	} else {
		if r.actuatorFailsafe {
			r.actuatorFailsafe = false
			log.Printf("[GREENHOUSE] Actuator recovered")
		}
		r.communicationOK = true
	}
	r.actuatorMutex.Unlock()

	return err
}

// GetCurrentHeaterPower implements ActuatorController interface
func (r *RealGreenhouseEnvironment) GetCurrentHeaterPower() float64 {
	r.actuatorMutex.RLock()
	defer r.actuatorMutex.RUnlock()
	return r.lastHeaterPower
}

// GetStatus returns detailed status information
func (r *RealGreenhouseEnvironment) GetStatus() map[string]interface{} {
	r.sensorMutex.RLock()
	sensorReading := r.lastSensorReading
	r.sensorMutex.RUnlock()

	r.actuatorMutex.RLock()
	heaterPower := r.lastHeaterPower
	failsafe := r.actuatorFailsafe
	r.actuatorMutex.RUnlock()

	return map[string]interface{}{
		"timestamp":             time.Now(),
		"internal_temp":         sensorReading.Temp.Internal,
		"internal_humidity":     sensorReading.Humidity.Internal,
		"external_temp":         sensorReading.Temp.External,
		"external_humidity":     sensorReading.Humidity.External,
		"heater_power":          heaterPower,
		"communication_ok":      r.communicationOK,
		"actuator_failsafe":     failsafe,
		"total_sensor_reads":    r.readCount,
		"total_actuator_writes": r.writeCount,
		"last_sensor_read":      r.lastReadTime,
		"last_actuator_write":   r.lastWriteTime,
	}
}

// EmergencyStop turns off all actuators immediately
func (r *RealGreenhouseEnvironment) EmergencyStop() error {
	log.Printf("[GREENHOUSE] EMERGENCY STOP triggered")
	err := r.hardwareActuators.EmergencyShutdown()

	r.actuatorMutex.Lock()
	r.lastHeaterPower = 0.0
	r.actuatorFailsafe = true
	r.actuatorMutex.Unlock()

	return err
}

// Close closes all hardware connections
func (r *RealGreenhouseEnvironment) Close() error {
	// Stop heater
	_ = r.hardwareActuators.SetHeaterPower(0.0)
	// TODO: set last needed?

	// Close hardware
	sensorErr := r.hardwareSensors.Close()
	actuatorErr := r.hardwareActuators.Close()

	if sensorErr != nil {
		return fmt.Errorf("sensor close error: %w", sensorErr)
	}
	if actuatorErr != nil {
		return fmt.Errorf("actuator close error: %w", actuatorErr)
	}

	return nil
}

// HealthCheck verifies hardware is responding
func (r *RealGreenhouseEnvironment) HealthCheck() error {
	// Try to read sensors
	_, err := r.ReadSensors()
	if err != nil {
		return fmt.Errorf("sensor health check failed: %w", err)
	}

	// Try to read actuator status
	_, err = r.hardwareActuators.GetHeaterStatus()
	if err != nil {
		return fmt.Errorf("actuator health check failed: %w", err)
	}

	return nil
}

// MockSensors provides a mock sensor implementation for testing without hardware
type MockSensors struct {
	internalTemp     float64
	internalHumidity float64
	externalTemp     float64
	externalHumidity float64
	errorRate        float64 // Probability of error (0.0-1.0)
}

// NewMockSensors creates mock sensors for testing
func NewMockSensors(internalTemp, internalHumidity, externalTemp, externalHumidity float64) *MockSensors {
	return &MockSensors{
		internalTemp:     internalTemp,
		internalHumidity: internalHumidity,
		externalTemp:     externalTemp,
		externalHumidity: externalHumidity,
		errorRate:        0.0,
	}
}

func (m *MockSensors) ReadTemperatureHumidity() (float64, float64, float64, float64, error) {
	return m.internalTemp, m.internalHumidity, m.externalTemp, m.externalHumidity, nil
}

func (m *MockSensors) Initialize() error {
	return nil
}

func (m *MockSensors) Close() error {
	return nil
}

// MockActuators provides a mock actuator implementation for testing without hardware
type MockActuators struct {
	heaterPower float64
}

// NewMockActuators creates mock actuators for testing
func NewMockActuators() *MockActuators {
	return &MockActuators{heaterPower: 0.0}
}

func (m *MockActuators) SetHeaterPower(power float64) error {
	m.heaterPower = power
	return nil
}

func (m *MockActuators) GetHeaterStatus() (float64, error) {
	return m.heaterPower, nil
}

func (m *MockActuators) Initialize() error {
	return nil
}

func (m *MockActuators) Close() error {
	return nil
}

func (m *MockActuators) EmergencyShutdown() error {
	m.heaterPower = 0.0
	return nil
}
