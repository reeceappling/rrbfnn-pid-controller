package coreSystem

import (
	"context"
	"fmt"
	"math"
	"time"
)

const (
	centerpointTemp       = 67.5
	centerpointInternalRH = 0.7  // TODO: validate ok
	centerpointExternalRH = 0.45 // TODO: validate ok
	normExternalTemp      = 17.5 // TODO: validate ok (probably correct)
	normInternalTemp      = 17.5 // TODO: validate ok (probably correct)
	normInternalRH        = 30.0 // TODO: validate ok
	normExternalRH        = 20.0 // TODO: validate ok

	normTempErr                = 5.0            // TODO: ensure ok (varies based on expected error?)
	normdErrTemp               = 1.0            // TODO: ensure ok
	normEIntTemp               = 1.0            // TODO: ensure ok
	normRHErr                  = 5.0            // TODO: ensure ok (varies based on expected error?)
	normdErrRH                 = 1.0            // TODO: ensure ok
	normEIntRH                 = 1.0            // TODO: ensure ok
	normEnvironmentDeltaTemp   = 10.0           // TODO: ensure ok
	normEnvironmentDeltaRH     = normInternalRH // TODO: ensure ok! SHOULD BE DIFFERENT!
	rhErrNormalizationConstant = 1.0            // TODO: fix?
	anotherNormalizer          = 1.0            // TODO: fix?
)

// Run executes the greenhouse control simulation
// If duration is 0, runs indefinitely until context is cancelled
func (cs *ControlSystem) Run(ctx context.Context, duration time.Duration) error {
	startTime := cs.Clock.Now()
	isPerpetual := duration == 0 // If duration is 0, run forever
	var endTime time.Time
	if !isPerpetual {
		endTime = startTime.Add(duration)
	}

	// For mock time, skip ticker and run as fast as possible
	mockClock, isMock := cs.Clock.(*MockClock)

	if isMock { // TODO: simplify? // TODO: put stats on a graph?
		// Fast simulation mode - no real delays
		for {
			if ctx.Err() != nil {
				return ctx.Err()
			}

			currentTime := mockClock.Now()
			if !isPerpetual && currentTime.After(endTime) {
				fmt.Println("Simulation complete")
				cs.Logger.Flush()
				return nil
			}

			// Check and apply setpoint changes
			if newSetpoint, exists := cs.SetpointChanges[currentTime.Sub(startTime)]; exists {
				cs.Setpoint = newSetpoint
				cs.PidCtrl.Reset() // Reset integral to avoid windup
				if cs.Verbose {
					fmt.Printf("⚡ Setpoint changed to %.1f°F at elapsed time %v\n", newSetpoint, currentTime.Sub(startTime))
				}
			}

			if err := cs.step(); err != nil { // TODO: can/should the previous step/error time be inserted here?
				return err
			}

			// Manually advance mock time
			mockClock.Advance(1 * time.Second)
		}
	} else {
		// Real-time simulation
		updateTicker := time.NewTicker(1 * time.Second)
		defer updateTicker.Stop()

		for {
			select {
			case <-ctx.Done():
				fmt.Println("Simulation stopped by context cancellation")
				cs.Logger.Flush()
				return ctx.Err()
			case <-updateTicker.C:
				currentTime := cs.Clock.Now()
				if !isPerpetual && currentTime.After(endTime) {
					fmt.Println("Simulation complete")
					cs.Logger.Flush()
					return nil
				}

				// Check and apply setpoint changes
				if newSetpoint, exists := cs.SetpointChanges[currentTime.Sub(startTime)]; exists {
					cs.Setpoint = newSetpoint
					cs.PidCtrl.Reset() // Reset integral to avoid windup
					if cs.Verbose {
						fmt.Printf("⚡ Setpoint changed to %.1f°F at elapsed time %v\n", newSetpoint, currentTime.Sub(startTime))
					}
				}

				if err := cs.step(); err != nil { // TODO: can/should the previous step/error time be inserted here?
					return err
				}
			}
		}
	}
}

// step executes one control iteration
func (cs *ControlSystem) step() error {
	// Check for immediate setpoint changes (thread-safe)
	cs.setpointMutex.Lock()
	if cs.immediateSetpoint != nil {
		cs.Setpoint = *cs.immediateSetpoint
		cs.immediateSetpoint = nil
		cs.PidCtrl.Reset() // Reset integral to avoid windup
		if cs.Verbose {
			fmt.Println(cs.pendingSetpointMsg)
		}
	}
	cs.setpointMutex.Unlock()

	// Get current greenhouse state from environment (simulator, real hardware, mock, etc.)
	state, err := cs.Environment.ReadSensors()
	if err != nil {
		if cs.Verbose {
			fmt.Printf("⚠ Sensor error: %v\n", err)
		}
		return err
	}

	// Compute tempErr
	tempErr := state.Temp.Internal - cs.Setpoint
	cs.ErrorHistory = append(cs.ErrorHistory, tempErr)

	// Compute humidity error
	rhErr := state.Humidity.Internal - cs.SetpointRH

	// TODO: if autosetting based on temp bounds, should the network be skipped?

	// Get RBF network output (adaptive PID gains)
	inputs := cs.constructNetworkInputs(state, tempErr, rhErr)

	// Sanity check: inputs should be exactly 41 dimensions
	if len(inputs) != 41 {
		return fmt.Errorf("inputs has %d dimensions, expected 41", len(inputs))
	}

	rbfOut := cs.RbfNetwork.Forward(inputs)

	// Run PID controller with adaptive gains
	// dt = 1 second (step is called once per second in both mock and real-time modes)
	dt := 1 * time.Second // TODO: I dont like this...
	pidOut := cs.PidCtrl.Compute(tempErr, rbfOut.Kp, rbfOut.Ki, rbfOut.Kd, dt)
	controlOut := ControlOutput{
		HeaterPower: pidOut.Output,
		Timestamp:   time.Now(),
	}
	// TODO: IF TEMPERATURE IS TOO LOW/high, override HEATER TO MAX/min?
	if cs.PidOverrider.TBound.OutOfBounds(state.Temp.Internal) {
		if state.Temp.Internal >= cs.PidOverrider.TBound.Max {
			controlOut.HeaterPower = 0.0
		} else {
			controlOut.HeaterPower = 1.0
		}
	} else {
		distToSp := cs.Setpoint - state.Temp.Internal
		if math.Abs(distToSp) > cs.PidOverrider.MaxTempDelta {
			if distToSp < 0 {
				controlOut.HeaterPower = 0.0
			} else {
				controlOut.HeaterPower = 1.0
			}
		}
	}

	// TODO: IF RH IS TOO LOW, override fogger TO MAX?
	// TODO: do rbfOut.PreSigmoidRaw need to be updated if this is the case?
	// Send control command to environment (simulator, real hardware, mock, etc.)
	if err := cs.Environment.SendControl(controlOut); err != nil {
		if cs.Verbose {
			fmt.Printf("⚠ Control error: %v\n", err)
		}
		return err
	}

	// Online learning: backprop and update network
	cs.RbfNetwork.BackwardAndUpdate(tempErr, rbfOut.HiddenActivations, rbfOut.AugmentedInputs, rbfOut.PreSigmoidRaw)

	// Adapt network structure
	cs.RbfNetwork.AdaptStructure(inputs, tempErr)

	// Record state in history buffer for next iteration's inputs
	cs.recordHistory(state)

	// Track statistics
	cs.totalLoss += tempErr * tempErr

	if cs.Verbose && len(cs.ErrorHistory)%100 == 0 {
		fmt.Printf("Step %d: Temp=%.1f°F, Setpoint=%.1f°F, Error=%.2f°F, Heater=%.3f, Kp=%.4f, Ki=%.4f, Kd=%.4f, Nodes=%d\n",
			len(cs.ErrorHistory),
			state.Temp.Internal,
			cs.Setpoint,
			tempErr,
			pidOut.Output,
			rbfOut.Kp, rbfOut.Ki, rbfOut.Kd,
			cs.RbfNetwork.GetNumNodes(),
		)
	}

	return nil
}

// recordHistory stores current state values in circular buffers for temporal input construction
func (cs *ControlSystem) recordHistory(state GreenhouseState) {
	cs.heaterHistory[cs.historyIndex] = state.HeaterPowerOutput
	cs.foggerHistory[cs.historyIndex] = state.FoggerPowerOutput
	cs.fanHistory[cs.historyIndex] = state.FanPowerOutput
	cs.tempHistory[cs.historyIndex] = state.Temp.Internal
	cs.humidityHistory[cs.historyIndex] = state.Humidity.Internal
	cs.historyIndex = (cs.historyIndex + 1) % cs.historySize
}

// getHistoricalValue retrieves a value from the circular buffer at the given offset (in steps, negative = past)
func (cs *ControlSystem) getHistoricalValue(buffer []float64, stepsAgo int) float64 {
	if stepsAgo < 0 {
		stepsAgo = 0
	}
	if stepsAgo >= cs.historySize {
		stepsAgo = cs.historySize - 1
	}
	if cs.historySize == 0 {
		panic("HISTORY SIZE WAS 0")
	}
	idx := (cs.historyIndex - 1 - stepsAgo + cs.historySize) % cs.historySize
	return buffer[idx]
}

// constructNetworkInputs builds the 41-dimensional input vector for the RBF network
func (cs *ControlSystem) constructNetworkInputsAny(state GreenhouseState, tempError float64, rhErr float64, doRH bool) []float64 {
	var pid *AdaptivePIDController
	var deltaEnv, normdErr, normInt float64
	if doRH {
		pid = cs.PidCtrlRH
		deltaEnv = (state.Humidity.Internal - state.Humidity.External) / normEnvironmentDeltaRH
		_, normdErr, normInt = normRHErr, normdErrRH, normEIntRH
	} else {
		pid = cs.PidCtrl
		deltaEnv = (state.Temp.Internal - state.Temp.External) / normEnvironmentDeltaTemp
		_, normdErr, normInt = normTempErr, normdErrTemp, normEIntTemp
	}
	inputs := make([]float64, 0, 41)

	// 5 historical time points (35 dimensions: 5 taps × 7 features)
	// Taps at: current (0s), 1 Min ago, 2 Min ago, 3 Min ago, 5 Min ago
	tapOffsets := [5]int{0, 60, 120, 180, 300}
	for i := 0; i < 5; i++ {
		offset := tapOffsets[i]
		inputs = append(inputs,
			cs.getHistoricalValue(cs.heaterHistory, offset),
			cs.getHistoricalValue(cs.foggerHistory, offset),
			cs.getHistoricalValue(cs.fanHistory, offset),
			(cs.getHistoricalValue(cs.tempHistory, offset)-centerpointTemp)/normInternalTemp,
			(state.Temp.External-centerpointTemp)/normExternalTemp,
			(cs.getHistoricalValue(cs.humidityHistory, offset)-centerpointInternalRH)/normInternalRH,
			(state.Humidity.External-centerpointExternalRH)/normExternalRH,
		)
	}

	// 6 additional dimensions // TODO: should probably be 5???
	// Compute derivative of error (dErr) from error history with actual dt
	dErr := 0.0
	if len(cs.ErrorHistory) > 1 {
		// dErr = (current - previous) for now; normalized by dt in PID controller
		prevErr := cs.ErrorHistory[len(cs.ErrorHistory)-2]
		dErr = tempError - prevErr // Will be normalized by dt=1s in PID Compute
	}
	eInt := pid.LastErrorIntegral()
	inputs = append(inputs,
		tempError/normTempErr, // Kept for RH?
		dErr/normdErr,
		eInt*pid.Ki()/normInt,
		rhErr/rhErrNormalizationConstant, // RH error
		deltaEnv,
		0.0/anotherNormalizer, // Reserve // TODO: ????
	)
	return inputs
}

// constructNetworkInputs builds the 41-dimensional input vector for the RBF network
func (cs *ControlSystem) constructNetworkInputsRH(state GreenhouseState, tempError float64, rhErr float64) []float64 {
	return cs.constructNetworkInputsAny(state, tempError, rhErr, true)
}

// constructNetworkInputs builds the 41-dimensional input vector for the RBF network
func (cs *ControlSystem) constructNetworkInputs(state GreenhouseState, tempError float64, rhErr float64) []float64 {
	return cs.constructNetworkInputsAny(state, tempError, rhErr, false)
}

// logData writes current state to CSV
// Note: This function is not currently used, but kept for future debugging
func (cs *ControlSystem) logData(state GreenhouseState) {
	tempError := state.Temp.Internal - cs.Setpoint

	gains := cs.PidCtrl.GetGains()

	cs.Logger.Log(LogEntry{
		Timestamp:        time.Now(),
		Setpoint:         cs.Setpoint,
		ActualTemp:       state.Temp.Internal,
		Error:            tempError,
		HeaterPower:      state.HeaterPowerOutput,
		Kp:               gains[0],
		Ki:               gains[1],
		Kd:               gains[2],
		ExternalTemp:     state.Temp.External,
		ExternalHumidity: state.Humidity.External,
		Humidity:         state.Humidity.Internal,
		Loss:             cs.totalLoss,
		NetworkNodeCount: cs.RbfNetwork.GetNumNodes(),
	})

	cs.lastLogged = time.Now()
}

// ChangeSetpoint schedules a setpoint change at a specific elapsed time
func (cs *ControlSystem) ChangeSetpoint(elapsedTime time.Duration, newSetpoint float64) {
	cs.SetpointChanges[elapsedTime] = newSetpoint
}

// SetSetpointImmediate changes the setpoint immediately (thread-safe)
// This is useful for real/perpetual greenhouses when using as an imported package
// The change takes effect at the next control step
func (cs *ControlSystem) SetSetpointImmediate(newSetpoint float64) {
	cs.setpointMutex.Lock()
	defer cs.setpointMutex.Unlock()
	cs.immediateSetpoint = &newSetpoint
	cs.pendingSetpointMsg = fmt.Sprintf("⚡ Setpoint changed to %.1f°F (immediate)", newSetpoint)
}

// GetSetpoint returns the current setpoint (thread-safe)
func (cs *ControlSystem) GetSetpoint() float64 {
	cs.setpointMutex.RLock()
	defer cs.setpointMutex.RUnlock()
	return cs.Setpoint
}

// ComputeStatistics calculates performance metrics
func (cs *ControlSystem) ComputeStatistics() Statistics {
	if len(cs.ErrorHistory) == 0 {
		return Statistics{}
	}

	stats := Statistics{}

	// Calculate basic error metrics
	sumAbsError, sumSqError, maxAbsError := 0.0, 0.0, 0.0

	for _, err := range cs.ErrorHistory {
		absErr := math.Abs(err)
		sumAbsError += absErr
		sumSqError += err * err

		if absErr > maxAbsError {
			maxAbsError = absErr
		}
	}

	n := float64(len(cs.ErrorHistory))
	stats.MeanAbsError = sumAbsError / n
	stats.RMSError = math.Sqrt(sumSqError / n)
	stats.MaxAbsError = maxAbsError

	// Steady-state error: average of final 5% of samples
	finalPercentageToUse := 0.05 // TODO: ensure ok
	steadyStateStart := len(cs.ErrorHistory) - int(n*finalPercentageToUse)
	if steadyStateStart < 0 {
		steadyStateStart = 0
	}

	steadyStateSumAbs := 0.0
	for i := steadyStateStart; i < len(cs.ErrorHistory); i++ {
		steadyStateSumAbs += math.Abs(cs.ErrorHistory[i])
	}
	steadyStateCount := len(cs.ErrorHistory) - steadyStateStart
	if steadyStateCount > 0 {
		stats.SteadyStateError = steadyStateSumAbs / float64(steadyStateCount)
	}

	stats.FinalNodeCount = cs.RbfNetwork.GetNumNodes()
	stats.TotalLoss = cs.totalLoss

	return stats
}
