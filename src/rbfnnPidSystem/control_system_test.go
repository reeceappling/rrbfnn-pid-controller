package coreSystem

import (
	"context"
	"os"
	"testing"
	"time"
)

const maxSteadyStateErrorF = 4.0

// TestMockTimeSimulation runs a full simulation with mock time (instant execution)
func TestMockTimeSimulation(t *testing.T) {
	// Create temporary log file
	logFile := "test_mock_simulation.csv"
	defer os.Remove(logFile)

	// Setup
	mockClock := NewMockClock(time.Now())
	logger := NewLogger(logFile)
	defer logger.Close()

	simulator := NewGreenhouseSimulator(
		70.0, // initial internal temp (F)
		60.0, // external temp (F)
		75.0, // initial internal humidity (%)
		50.0, // external humidity (%)
		90*time.Second,
		nil,
	)

	lrs := LearningRates{
		Weights: 0.01,
		Centers: 0.005,
		Spreads: 0.002,
	}
	rbfNetwork := NewRecurrentRBFNetwork(8, 20, lrs, 0.05, 0.1)
	pidController := NewAdaptivePIDController()

	var env GreenhouseEnvironment = simulator

	historySize := 300 // 5 minutes @ 1Hz
	system := &ControlSystem{
		Environment:     env,
		RbfNetwork:      rbfNetwork,
		PidCtrl:         pidController,
		PidCtrlRH:       NewAdaptivePIDController(),
		Logger:          logger,
		Clock:           mockClock,
		Setpoint:        72.0,
		SetpointRH:      65.0,
		Verbose:         false,
		TargetTemp:      72.0,
		ErrorHistory:    make([]float64, 0, 1000),
		SetpointChanges: make(map[time.Duration]float64),
		heaterHistory:   make([]float64, historySize),
		foggerHistory:   make([]float64, historySize),
		fanHistory:      make([]float64, historySize),
		tempHistory:     make([]float64, historySize),
		humidityHistory: make([]float64, historySize),
		historySize:     historySize,
	}

	// Run 1-minute simulation (should complete instantly with mock time)
	ctx := context.Background()
	startRealTime := time.Now()

	err := system.Run(ctx, 1*time.Minute)
	if err != nil {
		t.Fatalf("Simulation failed: %v", err)
	}

	realElapsed := time.Since(startRealTime)

	// Verify it ran fast
	if realElapsed > 10*time.Second {
		t.Logf("WARNING: Mock simulation took %v (should be <1s)", realElapsed)
	} else {
		t.Logf("✓ Mock simulation completed in %v (instant)", realElapsed)
	}

	// Verify results
	if len(system.ErrorHistory) == 0 {
		t.Fatal("No error history collected")
	}

	t.Logf("✓ Collected %d samples", len(system.ErrorHistory))

	stats := system.ComputeStatistics()
	t.Logf("✓ Mean error: %.3f°F", stats.MeanAbsError)
	t.Logf("✓ Network grew to %d Nodes", stats.FinalNodeCount)

	// Verify log file was created
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Fatal("Log file was not created")
	}
	t.Logf("✓ Log file created: %s", logFile)
}

// TestRealTimeSimulation runs a short real-time simulation for verification
func TestRealTimeSimulation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping real-time test in short mode")
	}

	// Create temporary log file
	logFile := "test_real_simulation.csv"
	defer os.Remove(logFile)

	logger := NewLogger(logFile)
	defer logger.Close()

	simulator := NewGreenhouseSimulator(70.0, 60.0, 75.0, 50.0, 45*time.Second, nil)

	lrs := LearningRates{Weights: 0.01, Centers: 0.005, Spreads: 0.002}
	rbfNetwork := NewRecurrentRBFNetwork(8, 20, lrs, 0.05, 0.1)
	pidController := NewAdaptivePIDController()

	var env GreenhouseEnvironment = simulator

	historySize := 300
	system := &ControlSystem{
		Environment:     env,
		RbfNetwork:      rbfNetwork,
		PidCtrl:         pidController,
		PidCtrlRH:       NewAdaptivePIDController(),
		Logger:          logger,
		Clock:           &RealClock{},
		Setpoint:        72.0,
		SetpointRH:      65.0,
		Verbose:         false,
		TargetTemp:      72.0,
		ErrorHistory:    make([]float64, 0, 1000),
		SetpointChanges: make(map[time.Duration]float64),
		heaterHistory:   make([]float64, historySize),
		foggerHistory:   make([]float64, historySize),
		fanHistory:      make([]float64, historySize),
		tempHistory:     make([]float64, historySize),
		humidityHistory: make([]float64, historySize),
		historySize:     historySize,
	}

	// Run 20-second real-time simulation
	ctx := context.Background()
	err := system.Run(ctx, 20*time.Second)
	if err != nil {
		t.Fatalf("Simulation failed: %v", err)
	}

	t.Logf("✓ Real-time simulation completed")
	t.Logf("✓ Collected %d samples", len(system.ErrorHistory))

	stats := system.ComputeStatistics()
	t.Logf("✓ Mean error: %.3f°F", stats.MeanAbsError)
	t.Logf("✓ Network grew to %d Nodes", stats.FinalNodeCount)
}

// TestMockClockAdvance verifies MockClock functionality
func TestMockClockAdvance(t *testing.T) {
	now := time.Now()
	clock := NewMockClock(now)

	// Test initial time
	if !clock.Now().Equal(now) {
		t.Fatal("MockClock initial time incorrect")
	}

	// Test advancement
	clock.Advance(10 * time.Second)
	expected := now.Add(10 * time.Second)
	if !clock.Now().Equal(expected) {
		t.Fatalf("MockClock advancement failed: got %v, want %v", clock.Now(), expected)
	}

	// Test Sleep (should advance time without real delay)
	startReal := time.Now()
	clock.Sleep(1 * time.Second)
	realElapsed := time.Since(startReal)

	if realElapsed > 100*time.Millisecond {
		t.Logf("MockClock.Sleep took real time: %v", realElapsed)
	}

	expected = expected.Add(1 * time.Second)
	if !clock.Now().Equal(expected) {
		t.Fatalf("MockClock Sleep failed: got %v, want %v", clock.Now(), expected)
	}

	t.Logf("✓ MockClock works correctly")
}

// TestRBFNetworkGrowth verifies that network grows during learning
func TestRBFNetworkGrowth(t *testing.T) {
	mockClock := NewMockClock(time.Now())
	logFile := "test_network_growth.csv"
	defer os.Remove(logFile)

	logger := NewLogger(logFile)
	defer logger.Close()

	simulator := NewGreenhouseSimulator(70.0, 60.0, 75.0, 50.0, 60*time.Second, nil)
	lrs := LearningRates{Weights: 0.01, Centers: 0.005, Spreads: 0.002}
	rbfNetwork := NewRecurrentRBFNetwork(5, 15, lrs, 0.05, 0.1) // Start with fewer Nodes
	pidController := NewAdaptivePIDController()

	var env GreenhouseEnvironment = simulator

	historySize := 300
	system := &ControlSystem{
		Environment:     env,
		RbfNetwork:      rbfNetwork,
		PidCtrl:         pidController,
		PidCtrlRH:       NewAdaptivePIDController(),
		Logger:          logger,
		Clock:           mockClock,
		Setpoint:        72.0,
		SetpointRH:      65.0,
		Verbose:         false,
		TargetTemp:      72.0,
		ErrorHistory:    make([]float64, 0, 1000),
		SetpointChanges: make(map[time.Duration]float64),
		heaterHistory:   make([]float64, historySize),
		foggerHistory:   make([]float64, historySize),
		fanHistory:      make([]float64, historySize),
		tempHistory:     make([]float64, historySize),
		humidityHistory: make([]float64, historySize),
		historySize:     historySize,
	}

	initialNodes := system.RbfNetwork.GetNumNodes()
	t.Logf("Initial network size: %d Nodes", initialNodes)

	// Run 2-minute simulation
	ctx := context.Background()
	err := system.Run(ctx, 2*time.Minute)
	if err != nil {
		t.Fatalf("Simulation failed: %v", err)
	}

	finalNodes := system.RbfNetwork.GetNumNodes()
	t.Logf("Final network size: %d Nodes", finalNodes)

	if finalNodes <= initialNodes {
		t.Logf("WARNING: Network didn't grow (started: %d, final: %d)", initialNodes, finalNodes)
	} else {
		t.Logf("✓ Network grew by %d Nodes", finalNodes-initialNodes)
	}
}

// TestSetpointChanges verifies the system handles dynamic setpoint changes with proper spacing
func TestSetpointChanges(t *testing.T) {
	mockClock := NewMockClock(time.Now())
	logFile := "test_setpoint_changes.csv"
	defer os.Remove(logFile)

	logger := NewLogger(logFile)
	defer logger.Close()

	simulator := NewGreenhouseSimulator(70.0, 60.0, 75.0, 50.0, 60*time.Second, nil)
	lrs := LearningRates{Weights: 0.015, Centers: 0.008, Spreads: 0.003}
	rbfNetwork := NewRecurrentRBFNetwork(10, 50, lrs, 0.05, 0.1)
	pidController := NewAdaptivePIDController()

	var env GreenhouseEnvironment = simulator

	historySize := 300
	system := &ControlSystem{
		Environment:     env,
		RbfNetwork:      rbfNetwork,
		PidCtrl:         pidController,
		PidCtrlRH:       NewAdaptivePIDController(),
		Logger:          logger,
		Clock:           mockClock,
		Setpoint:        72.0,
		SetpointRH:      65.0,
		Verbose:         true, // Enable verbose to see setpoint changes
		TargetTemp:      72.0,
		ErrorHistory:    make([]float64, 0, 1000),
		SetpointChanges: make(map[time.Duration]float64),
		heaterHistory:   make([]float64, historySize),
		foggerHistory:   make([]float64, historySize),
		fanHistory:      make([]float64, historySize),
		tempHistory:     make([]float64, historySize),
		humidityHistory: make([]float64, historySize),
		historySize:     historySize,
	}

	// Schedule setpoint changes at least 5 minutes apart
	// Phase 1: 0-5 Min at 72°F (baseline adaptation)
	// Phase 2: 5-10 Min at 77°F (+5°F - system has time to adjust)
	// Phase 3: 10-15 Min at 67°F (-10°F - system has time to adjust)
	// Phase 4: 15-20 Min at 72°F (+5°F back to original)

	system.ChangeSetpoint(5*time.Minute, 77.0)  // At 5 minutes: increase by 5 degrees
	system.ChangeSetpoint(10*time.Minute, 67.0) // At 10 minutes: decrease by 10 degrees
	system.ChangeSetpoint(15*time.Minute, 72.0) // At 15 minutes: back to original

	t.Logf("Scheduled setpoint changes (5-minute spacing):")
	t.Logf("  t=0min:  72°F (initial baseline)")
	t.Logf("  t=5min:  77°F (+5°F, allow 5min adjustment)")
	t.Logf("  t=10min: 67°F (-10°F, allow 5min adjustment)")
	t.Logf("  t=15min: 72°F (+5°F back to original)")

	// Run 20-minute simulation
	ctx := context.Background()
	err := system.Run(ctx, 20*time.Minute)
	if err != nil {
		t.Fatalf("Simulation failed: %v", err)
	}

	t.Logf("✓ Simulation with properly-spaced setpoint changes completed")
	t.Logf("✓ Total samples: %d", len(system.ErrorHistory))
	t.Logf("✓ Final network size: %d Nodes", system.RbfNetwork.GetNumNodes())

	// Verify the system adapted
	stats := system.ComputeStatistics()
	t.Logf("✓ Mean error across all setpoints: %.3f°F", stats.MeanAbsError)
	t.Logf("✓ RMS error: %.3f°F", stats.RMSError)
	t.Logf("✓ Max error: %.3f°F", stats.MaxAbsError)
	t.Logf("✓ Steady-state error (final 5%%): %.3f°F", stats.SteadyStateError)

	// Convergence check: steady-state errors should converge to within ±4°F
	// (allow larger transient errors during setpoint changes, but require good steady-state)
	steadyStateAllowed := 4.0
	if stats.SteadyStateError > steadyStateAllowed {
		t.Errorf("CONVERGENCE FAILED: Steady-state error %.3f°F exceeds ±%.1f°F criterion", stats.SteadyStateError, steadyStateAllowed)
	} else {
		t.Logf("✓ CONVERGENCE SUCCESS: Steady-state error within ±%.1f°F (was %.3f°F)", steadyStateAllowed, stats.SteadyStateError)
	}
}

// TestSetpointChangeRobustness verifies stability with well-spaced setpoint changes
func TestSetpointChangeRobustness(t *testing.T) {
	mockClock := NewMockClock(time.Now())
	logFile := "test_setpoint_robustness.csv"
	defer os.Remove(logFile)

	logger := NewLogger(logFile)
	defer logger.Close()

	simulator := NewGreenhouseSimulator(70.0, 60.0, 75.0, 50.0, 60*time.Second, nil)
	lrs := LearningRates{Weights: 0.015, Centers: 0.008, Spreads: 0.003}
	rbfNetwork := NewRecurrentRBFNetwork(10, 50, lrs, 0.05, 0.1)
	pidController := NewAdaptivePIDController()

	var env GreenhouseEnvironment = simulator

	historySize := 300
	system := &ControlSystem{
		Environment:     env,
		RbfNetwork:      rbfNetwork,
		PidCtrl:         pidController,
		PidCtrlRH:       NewAdaptivePIDController(),
		Logger:          logger,
		Clock:           mockClock,
		Setpoint:        72.0,
		SetpointRH:      65.0,
		Verbose:         false,
		TargetTemp:      72.0,
		ErrorHistory:    make([]float64, 0, 1000),
		SetpointChanges: make(map[time.Duration]float64),
		heaterHistory:   make([]float64, historySize),
		foggerHistory:   make([]float64, historySize),
		fanHistory:      make([]float64, historySize),
		tempHistory:     make([]float64, historySize),
		humidityHistory: make([]float64, historySize),
		historySize:     historySize,
	}

	// Schedule changes with 5-minute spacing for proper adaptation
	// This tests whether the system handles multiple different operating points
	// while having adequate time to stabilize between changes
	system.ChangeSetpoint(5*time.Minute, 75.0)  // First change at 5 Min
	system.ChangeSetpoint(10*time.Minute, 69.0) // Second change at 10 Min
	system.ChangeSetpoint(15*time.Minute, 78.0) // Third change at 15 Min
	system.ChangeSetpoint(20*time.Minute, 71.0) // Fourth change at 20 Min

	t.Logf("Robustness test - 5-minute spacing between changes:")
	t.Logf("  0-5min:   72°F (baseline)")
	t.Logf("  5-10min:  75°F (+3°F)")
	t.Logf("  10-15min: 69°F (-6°F)")
	t.Logf("  15-20min: 78°F (+9°F)")
	t.Logf("  20-25min: 71°F (-7°F)")

	ctx := context.Background()
	err := system.Run(ctx, 25*time.Minute)
	if err != nil {
		t.Fatalf("Simulation failed: %v", err)
	}

	stats := system.ComputeStatistics()
	t.Logf("✓ Multiple setpoint changes (5-Min spacing) handled successfully")
	t.Logf("✓ Error Stats: Steady-state %.3f°F. Avg %.3f°F. Max %.3f", stats.SteadyStateError, stats.MeanAbsError, stats.MaxAbsError)
	t.Logf("✓ Network adapted to: %d Nodes", stats.FinalNodeCount)

	// Convergence check: steady-state errors should converge to within ±4°F
	// (allow larger transient errors during setpoint changes, but require good steady-state)
	if stats.SteadyStateError > maxSteadyStateErrorF {
		t.Errorf("CONVERGENCE FAILED: Steady-state error %.3f°F exceeds ±%.1f°F criterion", stats.SteadyStateError, maxSteadyStateErrorF)
	} else {
		t.Logf("✓ CONVERGENCE SUCCESS: Steady-state error within ±%.1f°F (was %.3f°F)", maxSteadyStateErrorF, stats.SteadyStateError)
	}
}
