package pid

type ControlSignal struct {
	Output       float64 // Control output (0-1)
	ErrorTerm    float64 // Current error
	IntegralErr  float64 // Accumulated error
	DerivativErr float64 // Rate of change of error
}
type FloatBound struct {
	min, max float64
}

func newBound(min, max float64) FloatBound {
	return FloatBound{
		min: min,
		max: max,
	}
}
func (fc FloatBound) Clamp(val float64) (clampedValue float64, changed bool) {
	if val < fc.min {
		return fc.min, true
	}
	if val > fc.max {
		return fc.max, true
	}
	return val, false
}
