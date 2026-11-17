package coreSystem

import (
	"sync"
	"time"
)

// ControlOutput represents the control signals to send to greenhouse actuators
type ControlOutput struct {
	HeaterPower float64   // Heater power (0.0 = off, 1.0 = full) // TODO: needs fogger and fan?
	Timestamp   time.Time // When this command was issued
}

//type MultiControlOutput struct { // TODO: use???
//	HeaterPower float64   // Heater power (0.0 = off, 1.0 = full)
//	FoggerPower float64   // Fogger power (0.0 = off, 1.0 = full)
//	FanPower float64   // Fan power (0.0 = off, 1.0 = full)
//	Timestamp   time.Time // When this command was issued
//}

// GreenhouseEnvironment is a common interface for both simulated and real greenhouses
// This allows ControlSystem to work with any implementation (simulator, real hardware, mocks, etc.)
type GreenhouseEnvironment interface {
	// ReadSensors returns the current greenhouse state
	// Returns error if sensors are unavailable or outdated
	ReadSensors() (GreenhouseState, error)

	// SendControl sends a control command to the heater
	// powers should be in range [0, 1]
	SendControl(output ControlOutput) error

	// GetCurrentHeaterPower returns the last commanded heater power
	GetCurrentHeaterPower() float64

	// Close performs cleanup (close connections, turn off actuators, etc.)
	Close() error
}

// ControlSystem ties all components together
// Thread-Safety Note: SetSetpointImmediate() uses a RWMutex for thread-safe setpoint changes.
// All other operations assume single-threaded access during Run(). If calling SetSetpointImmediate()
// from another goroutine, ensure the calling goroutine doesn't interfere with simulation execution.
type ControlSystem struct {
	Environment     GreenhouseEnvironment // Supports any implementation (simulator, real HW, mock, etc.)
	RbfNetwork      *RecurrentRBFNetwork
	PidCtrl         *AdaptivePIDController
	PidOverrider    PidOverrider
	PidCtrlRH       *AdaptivePIDController // Humidity control PID
	Logger          *Logger
	Clock           Clock
	Setpoint        float64 // Temperature setpoint (°F)
	SetpointRH      float64 // Humidity setpoint (%)
	TargetTemp      float64
	Verbose         bool
	ErrorHistory    []float64
	totalLoss       float64
	lastLogged      time.Time
	SetpointChanges map[time.Duration]float64 // time -> new setpoint

	// History buffers for temporal inputs (5 minutes @ 1Hz = 300 samples)
	// These are circular buffers used to construct the 5-tap input vector
	heaterHistory   []float64
	foggerHistory   []float64
	fanHistory      []float64
	tempHistory     []float64
	humidityHistory []float64
	// TODO: airspeed history?
	historyIndex int
	historySize  int // Fixed at 300 for 5-minute window

	// Thread-safe setpoint change for real/perpetual greenhouses
	setpointMutex      sync.RWMutex
	immediateSetpoint  *float64 // nil = no immediate change pending
	pendingSetpointMsg string   // Message when setpoint changes
}

type PidOverrider struct {
	TBound       FloatBound
	MaxTempDelta float64
}

// LearningRates defines the RBF network learning rates
type LearningRates struct {
	Weights float64 // Weight learning rate (eta_w)
	Centers float64 // Center learning rate (eta_c)
	Spreads float64 // Spread (sigma) learning rate (eta_sigma)
}

// GreenhouseState represents current greenhouse conditions
type GreenhouseState struct {
	Time     time.Time        // TODO: replace with MultiControlOutput?
	Temp     InternalExternal // Temps (F)
	Humidity InternalExternal // Humidities (%)
	// TODO: windspeed?
	HeaterPowerOutput, FoggerPowerOutput, FanPowerOutput float64 // Current heater/fogger/fan powers (0-1)
}
type InternalExternal struct {
	Internal, External float64 // External is ambient outside of the greenhouse
}

// RBFOutput represents the RBF network output
type RBFOutput struct {
	Kp                float64   // Proportional gain
	Ki                float64   // Integral gain
	Kd                float64   // Derivative gain
	PreSigmoidRaw     []float64 // Raw outputs before sigmoid
	HiddenActivations []float64 // Current hidden layer activations
	AugmentedInputs   []float64 // Inputs + recurrent state
}

// ControlSignal represents the output from PID controller
type ControlSignal struct {
	Output       float64 // Control output (0-1)
	ErrorTerm    float64 // Current error
	IntegralErr  float64 // Accumulated error
	DerivativErr float64 // Rate of change of error
}

// MultiControlSignal represents the output from multiple PID controllers
type MultiControlSignal struct {
	HeaterControl, FoggerControl ControlSignal
}

// Statistics summarize control performance
type Statistics struct {
	MeanAbsError     float64
	RMSError         float64
	MaxAbsError      float64
	SteadyStateError float64
	FinalNodeCount   int
	TotalLoss        float64
	SettlingTime     time.Duration
}

// LogEntry represents a single logged data point
type LogEntry struct {
	Timestamp        time.Time
	Setpoint         float64
	ActualTemp       float64
	Error            float64
	HeaterPower      float64
	Kp               float64
	Ki               float64
	Kd               float64
	ExternalTemp     float64
	ExternalHumidity float64
	Humidity         float64
	Loss             float64
	NetworkNodeCount int
}

// TimestampedValue wraps a value with its timestamp
type TimestampedValue struct {
	Timestamp time.Time
	Value     float64
}

// DelayBuffer implements a fixed-size circular buffer for time-delayed inputs
type DelayBuffer struct {
	buffer []float64
	index  int
	size   int
	filled bool
}

// RBFNode represents a single RBF basis function center
type RBFNode struct {
	Center []float64 // Center point in input space
	/*
		diff = cat(inp,prevAct) - Center
		distSq = sum of squared diffs
		sigma := math.Max(net.Nodes[j].Spread, net.MinSpread*2) // Enforce minimum spread // TODO: why is this *2? should the other be /2 instead?
		sigma2 := sigma * sigma
		activations[j] = math.Exp(-distSq / (2.0 * sigma2))
	*/
	Spread float64 // Gaussian spread (sigma)
	/*
		actSpread = min(spread, minSpread)*/
	Activity float64 // EMA of activation (for pruning)
	/*
		net.Nodes[j].Activity = net.ActivityDecay*net.Nodes[j].Activity + (1.0-net.ActivityDecay)*activations[j]
	*/
}

// PIDState tracks the PID controller internal state
type PIDState struct {
	LastUpdate time.Time // LastErr update time
}

// PIDState tracks the PID controller internal state
type MultiPIDState struct {
	ErrorPrev     [2]float64 // Previous errors (for derivative)
	ErrorIntegral [2]float64 // Accumulated error integrals
	LastUpdate    time.Time  // LastErr update time
}
