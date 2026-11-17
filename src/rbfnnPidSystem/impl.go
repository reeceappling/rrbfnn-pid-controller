package coreSystem

var _ HardwareSensors = &RealSensors{}
var _ HardwareActuators = &RealActuators{}

type RealSensors struct{} // TODO: does this whole file need deleting or implementing?

func (r *RealSensors) ReadTemperatureHumidity() (float64, float64, float64, float64, error) {
	//TODO implement me
	panic("implement me")
}

func (r *RealSensors) Initialize() error {
	//TODO implement me
	panic("implement me")
}

func (r *RealSensors) Close() error {
	//TODO implement me
	panic("implement me")
}

type RealActuators struct{}

func (r *RealActuators) SetHeaterPower(power float64) error {
	//TODO implement me
	panic("implement me")
}

func (r *RealActuators) GetHeaterStatus() (float64, error) {
	//TODO implement me
	panic("implement me")
}

func (r *RealActuators) Initialize() error {
	//TODO implement me
	panic("implement me")
}

func (r *RealActuators) Close() error {
	//TODO implement me
	panic("implement me")
}

func (r *RealActuators) EmergencyShutdown() error {
	//TODO implement me
	panic("implement me")
}

// RealHeaterController demonstrates how to implement real actuator control
type RealHeaterController struct { // TODO: this!
	lastHeaterPower float64
}

// SendControl sends commands to real actuators
func (a *RealHeaterController) SendControl(output ControlOutput) error {
	//TODO implement me
	panic("implement me")
}

// GetCurrentHeaterPower returns the last commanded heater power
func (a *RealHeaterController) GetCurrentHeaterPower() float64 {
	return a.lastHeaterPower
}
