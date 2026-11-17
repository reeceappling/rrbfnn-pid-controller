package pid

type derivativeCalculator interface {
	calcDErr(dErr float64)
	Last() float64
	setErr(float64)
	combine() float64
	SetKd(float64)
	gain() float64
}
type derivativeCalculatorTypical struct {
	lastDErr float64
	Kd       float64
}

func (dc *derivativeCalculatorTypical) calcDErr(dErr float64) {
	dc.setDErr(dErr)
}
func (ic *derivativeCalculatorTypical) gain() float64 {
	return ic.Kd
}
func (dc *derivativeCalculatorTypical) Last() float64 {
	return dc.lastDErr
}
func (dc *derivativeCalculatorTypical) setDErr(dErr float64) {
	dc.lastDErr = dErr
}
func (dc *derivativeCalculatorTypical) combine() float64 {
	return dc.Kd * dc.lastDErr
}
func (dc *derivativeCalculatorTypical) SetKd(kd float64) {
	dc.Kd = kd
}

type derivativeCalculatorLowPassFilter struct {
	coreCalc *derivativeCalculatorTypical
	alpha    float64 // dErr low pass filter strength, higher = more responsive. (0,1)
}

func (dc derivativeCalculatorLowPassFilter) calcDErr(dErr float64) float64 {
	out := (dc.alpha * dErr) + ((1.0 - dc.alpha) * dc.coreCalc.lastDErr)
	dc.coreCalc.setDErr(out)
	return out
}
func (dc derivativeCalculatorLowPassFilter) LastErrPostFilter() float64 {
	return dc.coreCalc.Last()
}
func (dc derivativeCalculatorLowPassFilter) combine() float64 {
	return dc.coreCalc.combine()
}
