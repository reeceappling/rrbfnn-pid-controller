package coreSystem

// SensorReader interface abstracts sensor input (real or simulated)
// Deprecated: Use GreenhouseEnvironment interface instead for new code
type SensorReader interface {
	// ReadSensors returns the current greenhouse state
	ReadSensors() (GreenhouseState, error)
}

// ActuatorController interface abstracts actuator control (real or simulated)
// Deprecated: Use GreenhouseEnvironment interface instead for new code
type ActuatorController interface {
	// SendControl sends control commands to greenhouse actuators
	SendControl(power float64) error
	// GetCurrentHeaterPower returns the last commanded heater power
	GetCurrentHeaterPower() float64
}

// SimulatorSensorReader wraps the simulator to provide the SensorReader interface
type SimulatorSensorReader struct {
	simulator *GreenhouseSimulator
}

// NewSimulatorSensorReader creates a sensor reader from a simulator
func NewSimulatorSensorReader(simulator *GreenhouseSimulator) *SimulatorSensorReader {
	return &SimulatorSensorReader{simulator: simulator}
}

// ReadSensors reads the current greenhouse state from the simulator
func (s *SimulatorSensorReader) ReadSensors() (GreenhouseState, error) {
	state := s.simulator.GetState()
	return state, nil
}

// SimulatorActuatorController wraps the simulator to provide the ActuatorController interface
type SimulatorActuatorController struct {
	simulator       *GreenhouseSimulator
	lastHeaterPower float64
}

// NewSimulatorActuatorController creates an actuator controller for the simulator
func NewSimulatorActuatorController(simulator *GreenhouseSimulator) *SimulatorActuatorController {
	return &SimulatorActuatorController{
		simulator:       simulator,
		lastHeaterPower: 0.0,
	}
}

// SendControl sends a control command to the simulator
func (a *SimulatorActuatorController) SendControl(power float64) error {
	a.lastHeaterPower = power
	_ = a.simulator.Step(power)
	return nil
}

// GetCurrentHeaterPower returns the last commanded heater power
func (a *SimulatorActuatorController) GetCurrentHeaterPower() float64 {
	return a.lastHeaterPower
}

// Example: How to use SensorReader and ActuatorController with custom hardware
//
// For real hardware, implement these interfaces:
//   type RealSensors struct { /* hardware connections */ }
//   func (r *RealSensors) ReadSensors() (GreenhouseState, error) { /* read from hardware */ }
//
//   type RealActuators struct { /* hardware connections */ }
//   func (r *RealActuators) SendControl(power float64) error { /* send to hardware */ }
//   func (r *RealActuators) GetCurrentHeaterPower() float64 { /* read current power */ }
//
// Then use GreenhouseEnvironment interface instead (see real_greenhouse.go for full example)
