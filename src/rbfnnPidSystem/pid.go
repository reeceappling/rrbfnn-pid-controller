package coreSystem

import (
	"math"
	"time"
	"log"
	"encoding/json"
)

// AdaptivePIDController implements a standard PID controller with time-varying gains
type AdaptivePIDController struct {
	gains     [3]float64 //{Kp, Ki, Kd}
	workspace [3]float64 /* [3]float64{
		Current Error (or previous before set in Compute),
		Accumulated error integral
		dErr with noise filter
	}*/
	outputBounds FloatBound
	alpha        float64       // dErr low pass filter strength, higher = more responsive. (0,1)
	memoryTime   time.Duration // How long to remember integral error
	LastUpdate   time.Time
	minDtSec     float64
}

// NewAdaptivePIDController creates a new PID controller
func NewAdaptivePIDController() *AdaptivePIDController {
	return &AdaptivePIDController{
		outputBounds: newBound(0.0, 1.0),
		gains:        [3]float64{0.1, 0.01, 0.05},
		workspace:    [3]float64{0.0, 0.0, 0.0},
		memoryTime:   5 * time.Minute, // If changed, see PID_MEMORY_TIME.md and also change Ki inversely when it changes. Ki_new = Ki_old × (old_memory_time / new_memory_time)
		alpha:        0.5,             // TODO: used to be 0.3, trying out 0.5. If this performs worse, then revert
		LastUpdate:   time.Now(),
		minDtSec:     0.1, // TODO: is this ok?
	}
}

// TODO: use next line for importing pre-trained PIDs
func NewPIDFromJson(jsonBytes []byte) (*AdaptivePIDController, error) {
	out := &AdaptivePIDController{}
	err := json.Unmarshal(jsonBytes, out)
	return out, err
}
func (pid *AdaptivePIDController) ShowGains() [3]float64 { // TODO: USE!
	return pid.gains
}
func (pid *AdaptivePIDController) Kp() float64 {
	return pid.gains[0]
}
func (pid *AdaptivePIDController) Ki() float64 {
	return pid.gains[1]
}
func (pid *AdaptivePIDController) Kd() float64 {
	return pid.gains[2]
}
func (pid *AdaptivePIDController) LastErr() float64 {
	return pid.workspace[0]
}
func (pid *AdaptivePIDController) LastErrorIntegral() float64 {
	return pid.workspace[1]
}
func (pid *AdaptivePIDController) LastFilteredDErr() float64 {
	return pid.workspace[2]
}

// Compute executes one PID control step with adaptive gains
func (pid *AdaptivePIDController) Compute(error float64, Kp, Ki, Kd float64, dt time.Duration) ControlSignal {
	// Handle zero time step
	dtSec := dt.Seconds()
	if dtSec <= pid.minDtSec {
		log.Fatalf("dt provided to pid.Compute() (%.2f)was below the minimum of %.2f", dtSec, pid.minDtSec) // TODO: panic here, or what to do about dtSec being 0?
		//dtSec = pid.minDtSec
	}
	// Replacing values was benchmarked to be the fastest, per BenchmarkVectorReplacement
	pid.gains[0], pid.gains[1], pid.gains[2] = Kp, Ki, Kd

	// ========== Integral Term (with exponential forgetting) ==========
	// Integrate error with forgetting factor
	forgettingFactor := math.Exp(-dtSec / pid.memoryTime.Seconds())    // TODO: is this forgetting factor ok? Seems weighted
	newErrorIntegralBase := pid.LastErrorIntegral() * forgettingFactor // new error integral if the windup is capped
	newErrorIntegral := newErrorIntegralBase + (error * dtSec)         // new error integral if the windup is uncapped
	pid.workspace[1] = newErrorIntegral
	// Set on vector later after possible changes (removing the last term if out-of-bounds)
	//uI := Ki * pid.state.ErrorIntegral

	// ========== Derivative Term ===========
	// Use measurement derivative (not error derivative) to avoid derivative kick on setpoint changes
	// dE/dt approximation
	// MUST BE DONE BEFORE PROPORTIONAL BECAUSE PROPORTIONAL OVERWRITES LAST ERROR
	dError := (error - pid.LastErr()) / dtSec
	// Low-pass filter the derivative to reduce noise sensitivity
	// Simple exponential filter: dErr_filtered = (alpha * dErr_raw) + ((1-alpha) * dErr_filtered_prev)
	filteredDError := (pid.alpha * dError) + ((1.0 - pid.alpha) * pid.LastFilteredDErr()) // TODO: If error rates suffer, switch back LastFilteredDErr to LastErr
	pid.workspace[2] = filteredDError
	//uD := Kd * filteredDError

	// ========== Proportional Term ==========
	pid.workspace[0] = error

	// ========== Combine Terms ==========
	uP := pid.gains[0] * error
	uI := pid.gains[1] * newErrorIntegral
	uD := pid.gains[2] * filteredDError
	u := uP + uI + uD

	// ========== Output Saturation & Anti-Windup ==========
	var wasOutOfBounds = false
	if u, wasOutOfBounds = pid.outputBounds.Clamp(u); wasOutOfBounds {
		// Prevent integral windup
		pid.workspace[1] = newErrorIntegralBase
	}

	// Update state
	pid.LastUpdate = time.Now() // TODO: FIX THIS TO RELY ON dtSec????

	return ControlSignal{
		Output:       u,
		ErrorTerm:    error,
		IntegralErr:  newErrorIntegral,
		DerivativErr: filteredDError,
	}
}

// Reset clears the integral error (useful for setpoint changes)
func (pid *AdaptivePIDController) Reset() {
	pid.workspace[0] = 0.0
	pid.workspace[1] = 0.0
}

// GetGains returns the last used PID gains
func (pid *AdaptivePIDController) GetGains() [3]float64 {
	return pid.gains
}

//// GetState returns the current PID controller state
//func (pid *AdaptivePIDController) GetState() PIDState { // TODO: fix if needed
//	return pid.state
//}

//// AdaptivePIDControllerMat implements a standard PID controller with time-varying gains using matrices/vectors
//type AdaptivePIDControllerMat struct {
//	state PIDState
//
//	gains     *mat.VecDense // [3]float64{Kp, Ki, Kd}
//	workspace *mat.VecDense /* [3]float64{
//		Current Error (or previous before set in Compute),
//		Accumulated error integral
//		dErr with noise filter
//	}*/
//	outputBounds FloatBound
//	alpha        float64       // dErr low pass filter strength, higher = more responsive. (0,1)
//	memoryTime   time.Duration // How long to remember integral error
//}
//
//// NewAdaptivePIDController creates a new PID controller
//func NewAdaptivePIDControllerMat() *AdaptivePIDControllerMat {
//	defaultGains := []float64{0.1, 0.01, 0.05}
//	return &AdaptivePIDControllerMat{
//		state: PIDState{
//			LastUpdate: time.Now(),
//		},
//		outputBounds: newBound(0.0, 1.0),
//		gains:        mat.NewVecDense(3, defaultGains), // Default gains
//		workspace:    mat.NewVecDense(3, []float64{0.0, 0.0, 0.0}),
//		memoryTime:   5 * time.Minute, // If changed, see PID_MEMORY_TIME.md and also change Ki inversely when it changes. Ki_new = Ki_old × (old_memory_time / new_memory_time)
//		alpha:        0.5,
//	}
//}
//func (pid *AdaptivePIDControllerMat) Kp() float64 {
//	return pid.workspace.AtVec(0)
//}
//func (pid *AdaptivePIDControllerMat) Ki() float64 {
//	return pid.workspace.AtVec(1)
//}
//func (pid *AdaptivePIDControllerMat) Kd() float64 {
//	return pid.workspace.AtVec(2)
//}
//func (pid *AdaptivePIDControllerMat) LastErr() float64 {
//	return pid.workspace.AtVec(0)
//}
//func (pid *AdaptivePIDControllerMat) LastErrorIntegral() float64 {
//	return pid.workspace.AtVec(1)
//}
//func (pid *AdaptivePIDControllerMat) LastFilteredDErr() float64 {
//	return pid.workspace.AtVec(2)
//}
//
//// Compute executes one PID control step with adaptive gains
//func (pid *AdaptivePIDControllerMat) Compute(error float64, Kp, Ki, Kd float64, dt time.Duration) ControlSignal {
//	// Handle zero time step
//	dtSec := dt.Seconds()
//	if dtSec <= 0 {
//		dtSec = 0.1 // TODO: panic here, or what to do about dtSec being 0?
//	}
//	lastErr, lastErrorIntegral := pid.LastErr(), pid.LastErrorIntegral()
//	// Replacing values was benchmarked to be the fastest, per BenchmarkVectorReplacement
//	pid.gains.SetVec(0, Kp)
//	pid.gains.SetVec(1, Ki)
//	pid.gains.SetVec(2, Kd)
//
//	// ========== Proportional Term ==========
//
//	pid.workspace.SetVec(0, error)
//	//uP := Kp * error
//
//	// ========== Integral Term (with exponential forgetting) ==========
//	// Integrate error with forgetting factor
//	forgettingFactor := math.Exp(-dtSec / pid.memoryTime.Seconds()) // TODO: is this forgetting factor ok? Seems weighted
//	newErrorIntegralBase := lastErrorIntegral * forgettingFactor
//	newErrorIntegral := newErrorIntegralBase + (error * dtSec)
//	// Set on vector later after possible changes (removing the last term if out-of-bounds)
//	//uI := Ki * pid.state.ErrorIntegral
//
//	// ========== Derivative Term ==========
//	// Use measurement derivative (not error derivative) to avoid derivative kick on setpoint changes
//	// dE/dt approximation
//	dError := (error - lastErr) / dtSec
//	alphaVec := mat.NewVecDense(2, []float64{pid.alpha, 1.0 - pid.alpha})
//	secondTerm := pid.LastFilteredDErr() // TODO: was lastErr
//	dErrLastErrVec := mat.NewVecDense(2, []float64{dError, secondTerm})
//	// Low-pass filter the derivative to reduce noise sensitivity
//	// Simple exponential filter: dErr_filtered = (alpha * dErr_raw) + ((1-alpha) * dErr_filtered_prev)
//	filteredDError := mat.Dot(alphaVec, dErrLastErrVec) // TODO: lastErr might not be right here? maybe use the last dErr?
//	pid.workspace.SetVec(2, filteredDError)
//	//uD := Kd * filteredDError
//
//	// ========== Combine Terms ==========
//	u := mat.Dot(pid.gains, pid.workspace)
//	//u := uP + uI + uD
//
//	// ========== Output Saturation & Anti-Windup ==========
//	if uModified, isOutOfBounds := pid.outputBounds.ClampWithOOB(u); isOutOfBounds {
//		u = uModified
//		// Prevent integral windup
//		newErrorIntegral = newErrorIntegralBase
//	}
//	pid.workspace.SetVec(1, newErrorIntegral)
//
//	// Update state
//	pid.state.LastUpdate = time.Now()
//
//	return ControlSignal{
//		Output:       u,
//		ErrorTerm:    error,
//		IntegralErr:  newErrorIntegral,
//		DerivativErr: filteredDError,
//	}
//}
//
//// Reset clears the integral error (useful for setpoint changes)
//func (pid *AdaptivePIDControllerMat) Reset() {
//	pid.workspace.SetVec(0, 0.0)
//	pid.workspace.SetVec(1, 0.0)
//}
//
//// GetGains returns the last used PID gains
//func (pid *AdaptivePIDControllerMat) GetGains() (out [3]float64) {
//	for i, val := range pid.gains.RawVector().Data {
//		out[i] = val
//	}
//	return out
//}
//
//// GetState returns the current PID controller state
//func (pid *AdaptivePIDControllerMat) GetState() PIDState {
//	return pid.state
//}

//
//// TODO: UTILIZE MULTIPID!
//// MultiPIDController implements a standard PID controller with time-varying gains
//type MultiPIDController struct {
//	gains     *mat.Dense // [2][3]float64{Kp, Ki, Kd}
//	workspace *mat.Dense /* [2][3]float64{
//		Temp{
//			Current Error (or previous before set in Compute),
//			Accumulated error integral
//			dErr with noise filter
//		}
//		RH{same}
//
//	}*/
//	PIDState
//	outputBounds FloatBound
//	alpha        float64       // dErr low pass filter strength, higher = more responsive. (0,1)
//	memoryTime   time.Duration // How long to remember integral error
//}

//// NewMultiPIDController creates a new PID controller
//func NewMultiPIDController() *MultiPIDController {
//	return &MultiPIDController{
//		gains: mat.NewDense(2, 3, []float64{
//			0.1, 0.01, 0.05, // Temp Default Gains
//			0.1, 0.01, 0.05, // RH Default Gains // TODO; ensure ok
//		}),
//		workspace: mat.NewDense(2, 3, []float64{
//			0.0, 0.0, 0.0, 0.0, 0.0, 0.0,
//		}),
//		PIDState: PIDState{
//			LastUpdate: time.Now(),
//		},
//		outputBounds: newBound(0.0, 1.0),
//		memoryTime:   5 * time.Minute, // If changed, see PID_MEMORY_TIME.md and also change Ki inversely when it changes. Ki_new = Ki_old × (old_memory_time / new_memory_time)
//	}
//}

//// Compute executes one PID control step with adaptive gains on both PID controllers
//func (pid *MultiPIDController) Compute(errorT, errorRH, KpT, KiT, KdT, KpRH, KiRH, KdRH float64, dt time.Duration) MultiControlSignal {
//	// Handle zero time step
//	dtSec := dt.Seconds()
//	if dtSec <= 0 {
//		dtSec = 0.1 // TODO: ok?
//	}
//	forgettingFactor := math.Exp(-dtSec / pid.memoryTime.Seconds()) // TODO: is this forgetting factor ok? Seems weighted
//
//	pid.gains.SetRow(0, []float64{KpT, KiT, KdT})
//	pid.gains.SetRow(1, []float64{KpRH, KiRH, KdRH})
//	errors := mat.NewVecDense(2, []float64{errorT, errorRH})
//	alphaVec := mat.NewVecDense(2, []float64{pid.alpha, 1.0 - pid.alpha})
//	wg := &sync.WaitGroup{}
//	wg.Add(2)
//	out := MultiControlSignal{}
//	// Get PrevVals
//	prevErr := pid.workspace.ColView(0)           // TODO: ensure ok
//	lastErrorIntegral := pid.workspace.ColView(1) // TODO: ensure ok
//	lastFilteredDErr := pid.workspace.ColView(2)
//	dError := mat.NewDense(2, 1, []float64{0.0, 0.0})
//	dError.Sub(errors, prevErr)
//	dError.Scale(1/dtSec, dError)
//	filteredDError := mat.NewVecDense(2, []float64{0.0, 0.0})
//	// Calculate new vals
//	newErrorIntegralBase := mat.NewVecDense(2, []float64{0.0, 0.0})
//	newErrorIntegral := mat.NewVecDense(2, []float64{0.0, 0.0})
//	newErrorIntegralBase.ScaleVec(forgettingFactor, lastErrorIntegral)
//	newErrorIntegral.ScaleVec(dtSec, errors)
//	newErrorIntegral.AddVec(newErrorIntegralBase, newErrorIntegral)
//	// TODO: copy new error into workspace!
//	pid.workspace.SetCol(0, errors.RawVector().Data)
//	// TODO: copy new error integral into workspace!
//	pid.workspace.SetCol(1, newErrorIntegral.RawVector().Data)
//
//	temp := mat.NewDense(2, 2, []float64{0.0, 0.0, 0.0, 0.0})
//	temp.SetCol(0, dError.RawMatrix().Data) // TODO: ensure ok
//	for i := 0; i < lastFilteredDErr.Len(); i++ {
//		temp.Set(1, i, lastFilteredDErr.At(0, i)) // TODO: ensure ok
//	}
//	alphaVec.MulVec(filteredDError.T(), alphaVec) // TODO: unsure if ok. Might not need transpose!
//	pid.workspace.SetCol(1, alphaVec.RawVector().Data)
//	// TODO: copy new error delta into workspace!
//	us := mat.NewDense(2, 1, []float64{0.0, 0.0}) // TODO: ensure dims ok
//	us.Mul(pid.gains, pid.workspace.T())          // TODO: any transposes?
//	// TODO: ensure Us are properly capped. If they are capped, then wind down the error integral for that case!
//	go func() {
//		prevErr := pid.workspace.At(0, 0)
//		// ========== Proportional Term ==========
//		uPT := KpT * errorT
//
//		// ========== Integral Term (with exponential forgetting) ==========
//		// Integrate error with forgetting factor
//		lastErrorIntegral := pid.workspace.At(0, 1)
//		newErrorIntegralBase := lastErrorIntegral * forgettingFactor
//		newErrorIntegral := newErrorIntegralBase + errorT*dtSec
//		uIT := KiT * pid.ErrorIntegral[0]
//
//		// ========== Derivative Term ==========
//		// Use measurement derivative (not error derivative) to avoid derivative kick on setpoint changes
//		// dE/dt approximation
//		dError := (errorT - prevErr) / dtSec
//
//		// Low-pass filter the derivative to reduce noise sensitivity
//		// Simple exponential filter: dErr_filtered = alpha * dErr_raw + (1-alpha) * dErr_filtered_prev
//		alpha := 0.3 // Filter strength (higher = more responsive) // TODO: is this ok?
//		filteredDErrorT := alpha*dError + (1.0-alpha)*pid.workspace.At(0, 2)
//
//		uDT := KdT * filteredDErrorT
//
//		// ========== Combine Terms ==========
//		uT := uPT + uIT + uDT
//
//		// ========== Output Saturation & Anti-Windup ==========
//		if uTTemp, clamped := pid.outputBounds.ClampWithOOB(uT); clamped {
//			uT = uTTemp
//			newErrorIntegral -= errorT * dtSec
//		}
//		pid.workspace.Set(0, 1, newErrorIntegral)
//
//		// Update state
//		pid.ErrorPrev[0] = errorT
//		out.HeaterControl = ControlSignal{
//			Output:       uT,
//			ErrorTerm:    errorT,
//			IntegralErr:  newErrorIntegral,
//			DerivativErr: filteredDErrorT,
//		}
//	}()
//	go func() {
//		// ========== Proportional Term ==========
//		uPRH := KpRH * errorRH
//
//		// ========== Integral Term (with exponential forgetting) ==========
//		// Integrate error with forgetting factor
//		pid.ErrorIntegral[1] = pid.ErrorIntegral[1]*forgettingFactor + errorRH*dtSec
//		uIRH := KiRH * pid.ErrorIntegral[1]
//
//		// ========== Derivative Term ==========
//		// Use measurement derivative (not error derivative) to avoid derivative kick on setpoint changes
//		// dE/dt approximation
//		dErrorRH := (errorRH - pid.ErrorPrev[1]) / dtSec
//
//		// Low-pass filter the derivative to reduce noise sensitivity
//		// Simple exponential filter: dErr_filtered = alpha * dErr_raw + (1-alpha) * dErr_filtered_prev
//		alpha := 0.3 // Filter strength (higher = more responsive) // TODO: is this ok?
//		filteredDErrorRH := alpha*dErrorRH + (1.0-alpha)*pid.ErrorPrev[1]
//
//		uDRH := KdRH * filteredDErrorRH
//
//		// ========== Combine Terms ==========
//		uRH := uPRH + uIRH + uDRH
//
//		// ========== Output Saturation & Anti-Windup ==========
//		if uRH > pid.maxOutput {
//			uRH = pid.maxOutput
//			// Prevent integral windup
//			pid.ErrorIntegral[1] -= errorRH * dtSec
//		} else if uRH < pid.minOutput {
//			uRH = pid.minOutput
//			pid.ErrorIntegral[1] -= errorRH * dtSec
//		}
//
//		// Update state
//		pid.ErrorPrev[1] = errorRH
//		out.HeaterControl = ControlSignal{
//			Output:       uRH,
//			ErrorTerm:    errorRH,
//			IntegralErr:  pid.ErrorIntegral[1],
//			DerivativErr: filteredDErrorRH,
//		}
//	}()
//	wg.Wait()
//	pid.lastGains = [6]float64{KpT, KiT, KdT, KpRH, KiRH, KdRH}
//	pid.LastUpdate = time.Now()
//
//	return out
//}
//
//// Reset clears the integral error (useful for setpoint changes)
//func (pid *MultiPIDController) Reset() {
//	pid.ErrorPrev = [2]float64{0.0, 0.0}
//	pid.ErrorIntegral = [2]float64{0.0, 0.0}
//}
//
//// GetGains returns the last used PID gains
//func (pid *MultiPIDController) GetGains() [6]float64 {
//	return pid.lastGains
//}
//
//// GetState returns the current PID controller state
//func (pid *MultiPIDController) GetState() MultiPIDState {
//	return pid.MultiPIDState
//}
