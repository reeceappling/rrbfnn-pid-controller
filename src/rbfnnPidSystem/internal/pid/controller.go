package pid

import (
	"time"
	"log"
)

type Controller struct {
	output       float64
	proportional proportionalCalculator
	integral     integralCalculator
	derivative   derivativeCalculator
	outputBounds *FloatBound
	LastUpdate   time.Time
	minDtSec     float64
}

func NewController(pCalc proportionalCalculator, iCalc integralCalculator, dCalc derivativeCalculator, minDtSec *float64, outputBounds *FloatBound) *Controller {
	actualMinDtSec := 0.1 // TODO: ok?
	if minDtSec != nil {
		actualMinDtSec = *minDtSec
	}
	return &Controller{
		proportional: pCalc,
		integral:     iCalc,
		derivative:   dCalc,
		outputBounds: outputBounds,
		LastUpdate:   time.Now(),
		minDtSec:     actualMinDtSec,
	}
}
func (pid *Controller) setGains(Kp, Ki, Kd float64) {
	pid.proportional.SetKp(Kp)
	pid.integral.SetKi(Ki)
	pid.derivative.SetKd(Kd)
}

// Compute executes one PID control step with adaptive gains
func (pid *Controller) Compute(error float64, Kp, Ki, Kd float64, dt time.Duration) ControlSignal {
	// Handle zero time step
	dtSec := dt.Seconds()
	if dtSec <= pid.minDtSec {
		log.Fatalf("dt provided to pid.Compute() (%.2f)was below the minimum of %.2f", dtSec, pid.minDtSec) // TODO: panic here, or what to do about dtSec being 0?
	}
	pid.setGains(Kp, Ki, Kd)

	// ========== Integral Term ==========
	newErrorIntegralBase, _ := pid.integral.calculate(error, dtSec)

	// ========== Derivative Term ===========
	dErr := (error - pid.proportional.LastErr()) / dtSec
	pid.derivative.calcDErr(dErr)

	// ========== Proportional Term ==========
	// Must be done after derivative because derivative relies on previous error
	pid.proportional.calculate(error)

	// ========== Combine Terms ==========
	outputValue := pid.calculateOutput()

	// ========== Output Saturation & Anti-Windup ==========
	var wasOutOfBounds = false
	if pid.output, wasOutOfBounds = pid.outputBounds.Clamp(outputValue); wasOutOfBounds {
		// Prevent integral windup
		pid.integral.SetLast(newErrorIntegralBase)
	}

	// Update state
	pid.LastUpdate = time.Now() // TODO: FIX THIS TO RELY ON dtSec????
	return pid.controlSignal()
}

func (pid *Controller) calculateOutput() float64 {
	out := pid.uP() + pid.uI() + pid.uD()
	pid.output = out
	return out
}
func (pid *Controller) uP() float64 {
	return pid.proportional.combine()
}
func (pid *Controller) uI() float64 {
	return pid.integral.combine()
}
func (pid *Controller) uD() float64 {
	return pid.derivative.combine()
}
func (pid *Controller) controlSignal() ControlSignal {
	return ControlSignal{
		Output:       pid.output,
		ErrorTerm:    pid.proportional.LastErr(),
		IntegralErr:  pid.integral.lastValue(),
		DerivativErr: pid.derivative.Last(),
	}
}

// Reset clears the integral error (useful for setpoint changes)
func (pid *Controller) Reset() {
	pid.proportional.SetErr(0)
	pid.integral.SetLast(0)
}

// GetGains returns the last used PID gains
func (pid *Controller) GetGains() (Kp, Ki, Kd float64) {
	return pid.proportional.gain(),
		pid.integral.gain(),
		pid.derivative.gain()
}
