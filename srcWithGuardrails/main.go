package srcWithGuardrails

//import (
//	"time"
//	"math"
//)

///*
//If temp is below A, or setPoint-currentTemp>B, Max out the heater?
//*/
//type EnvironmentAwareHeaterActuator struct {
//	maxTempDelta, minTemp, maxTemp float64
//}
//type EnvironmentAwareFogger struct {
//	maxRHDelta, minRH, maxRH float64
//}
//type DeltaBounds interface {
//	boundFor(val float64, setPoint float64, highOutput, lowOutput, actualOutput float64) float64
//}
//type SymmetricBounds struct {
//	maxDelta Delta
//}
//
//func (sb SymmetricBounds) boundFor(val float64, setPoint float64, highVal, lowVal, actualVal float64) float64 {
//	if val >= sb.maxDelta.higherBound(setPoint) {
//		return highVal
//	}
//	if val <= sb.maxDelta.lowerBound(setPoint) {
//		return lowVal
//	}
//	return actualVal
//}
//
//type AsymmetricBounds struct {
//	lowerDelta, upperDelta Delta
//}
//
//func (sb AsymmetricBounds) boundFor(val float64, setPoint float64, highVal, lowVal, actualVal float64) float64 {
//	if val >= sb.upperDelta.higherBound(setPoint) {
//		return highVal
//	}
//	if val <= sb.lowerDelta.lowerBound(setPoint) {
//		return lowVal
//	}
//	return actualVal
//}
//
//type Delta interface {
//	higherBound(val float64) float64
//	lowerBound(val float64) float64
//}
//type PctDelta float64
//
//func (pd PctDelta) higherBound(val float64) float64 {
//	return val * (1.0 + float64(pd))
//}
//func (pd PctDelta) lowerBound(val float64) float64 {
//	return val * (1.0 - float64(pd))
//}
//
//type StdDelta float64
//
//func (pd StdDelta) higherBound(val float64) float64 {
//	return val + float64(pd)
//}
//func (pd StdDelta) lowerBound(val float64) float64 {
//	return val - float64(pd)
//}
//
//func (ha EnvironmentAwareHeaterActuator) calcPower(error, Kp, Ki, Kd float64, dt time.Duration, currentTemp float64, setPointTemp float64, c *AdaptivePIDController) ControlSignal {
//	return calcActuatorPower(error, Kp, Ki, Kd, dt, currentTemp, setPointTemp, ha.maxTempDelta, ha.minTemp, ha.maxTemp, c)
//}
//
////func calcHeaterPower(error, Kp, Ki, Kd float64, dt time.Duration, currentTemp float64, setPointTemp float64, maxTempDelta float64, minTemp, maxTemp float64, c *AdaptivePIDController)(ControlSignal){
////	return calcActuatorPower(error, Kp, Ki, Kd, dt, currentTemp, setPointTemp, maxTempDelta, minTemp, maxTemp, c)
////}
//
//func calcActuatorPower(error, Kp, Ki, Kd float64, dt time.Duration, currentEnv float64, setPointEnv float64, maxEnvDelta float64, minEnv, maxEnv float64, c *AdaptivePIDController) ControlSignal {
//	cs := c.Compute(error, Kp, Ki, Kd, dt)
//	tempDelta := math.Abs(setPointEnv - currentEnv)
//	if currentEnv < setPointEnv {
//		if tempDelta > maxEnvDelta || currentEnv <= minEnv {
//			cs.Output = 1.0
//		}
//	} else {
//		if tempDelta > maxEnvDelta || currentEnv >= maxEnv {
//			cs.Output = 0.0
//		}
//	}
//	return cs
//}
//
//type ActuatorController interface {
//	SetOutput(val float64) error     // [0,1.0]
//	CurrentOutput() (float64, error) // [0,1.0]
//}
//
//var zeroOneClamper = func(val float64) float64 {
//	if val <= 0 {
//		return 0
//	}
//	if val >= 1 {
//		return 1
//	}
//	return val
//}
//
//type ActuatorWrapper struct {
//	Actuator ActuatorController
//}
//
//func (aw ActuatorWrapper) setOutput(val float64) error {
//	return aw.Actuator.SetOutput(zeroOneClamper(val))
//}
//func (aw ActuatorWrapper) currentOutput() (float64, error) {
//	return aw.Actuator.CurrentOutput()
//}
