package pid

type proportionalCalculator interface {
	LastErr() float64
	calculate(newErr float64)
	combine() float64
	SetKp(float64)
	SetErr(float64)
	gain() float64
}
type proportionalCalcImpl struct {
	lastErr float64
	Kp      float64
}

func (ic *proportionalCalcImpl) LastErr() float64 {
	return ic.lastErr
}
func (ic *proportionalCalcImpl) gain() float64 {
	return ic.Kp
}
func (ic *proportionalCalcImpl) SetErr(val float64) {
	ic.lastErr = val
}
func (ic *proportionalCalcImpl) calculate(newErr float64) {
	ic.SetErr(newErr)
}
func (ic *proportionalCalcImpl) combine() float64 {
	return ic.Kp * ic.lastErr
}
func (ic *proportionalCalcImpl) SetKp(kp float64) {
	ic.Kp = kp
}
