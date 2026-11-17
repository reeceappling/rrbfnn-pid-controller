package pid

import (
	"math"
	"time"
)

type integralCalculator interface {
	SetLast(float64)
	lastValue() float64
	calculate(newErr, dtSec float64) (uncapped, capped float64)
	combine() float64
	SetKi(float64)
	gain() float64
}
type intCalcTypical struct {
	lastIntegral float64
	Ki           float64
}
type intCalcForgetLinear struct {
	*intCalcTypical
	memoryTime time.Duration // How long to remember integral error
}
type intCalcForgetExponential struct {
	*intCalcTypical
	memoryTime time.Duration // How long to remember integral error
}

func (ic *intCalcTypical) lastValue() float64 {
	return ic.lastIntegral
}
func (ic *intCalcTypical) calculate(newErr, dtSec float64) (uncapped, capped float64) {
	newErrorIntegral := ic.lastIntegral + (newErr * dtSec) // new error integral if the windup is uncapped
	return newErrorIntegral, newErrorIntegral
}
func (ic *intCalcTypical) combine() float64 {
	return ic.Ki * ic.lastIntegral
}
func (ic *intCalcTypical) SetKi(ki float64) {
	ic.Ki = ki
}
func (ic *intCalcTypical) gain() float64 {
	return ic.Ki
}
func (ic *intCalcTypical) SetLast(val float64) {
	ic.lastIntegral = val
}
func (ic intCalcForgetLinear) calculate(newErr, dtSec float64) (uncapped, capped float64) {
	forgettingFactor := dtSec / ic.memoryTime.Seconds()
	// TODO: ensure forgettingFactor is utilized correctly
	newErrorIntegralBase := ic.lastIntegral * (1.0 - forgettingFactor) // new error integral if the windup is capped
	newErrorIntegral := ic.lastIntegral + (newErr * forgettingFactor)  // new error integral if the windup is uncapped
	ic.lastIntegral = newErrorIntegral
	return newErrorIntegralBase, newErrorIntegral
}
func (ic intCalcForgetExponential) calcIntegral(newErr, dtSec float64) (uncapped, capped float64) {
	forgettingFactor := math.Exp(-dtSec / ic.memoryTime.Seconds())
	newErrorIntegralBase := ic.lastIntegral * forgettingFactor // new error integral if the windup is capped
	newErrorIntegral := ic.lastIntegral + (newErr * dtSec)     // new error integral if the windup is uncapped
	ic.lastIntegral = newErrorIntegral
	return newErrorIntegralBase, newErrorIntegral
}
