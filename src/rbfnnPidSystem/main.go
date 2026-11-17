package coreSystem

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"time"
)

const (
	learningRateWeights     = 0.015
	learningRateCenters     = 0.008
	learningRateSpreads     = 0.003
	initialHiddenNodes      = 10
	maxHiddenNodes          = 50
	numErrorHistory         = 1000
	initialInternalTemp     = 70.0
	initialExternalTemp     = 60.0
	initialInternalHumidity = 75.0
	initialExternalHumidity = 50.0
	initialDefaultSetpointF = 72.0
)

type SimConfig struct {
	Duration time.Duration
	Setpoint float64
	LogFile  string
	//HeaterDelay                                       time.Duration
	Verbose, UseMockTime, EnableGrowth, EnablePruning bool
	RngSeed                                           int64
}

func RunRealGreenhouse(ctx context.Context, config SimConfig) error {
	// Example: To use real hardware, implement HardwareSensors and HardwareActuators interfaces
	// then instantiate: RealGreenhouseEnvironment with your implementations
	sensors := NewMockSensors(70.0, 75.0, 60.0, 50.0)
	actuators := NewMockActuators()
	gh, err := NewRealGreenhouseEnvironment(sensors, actuators, 5*time.Second, 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to initialize greenhouse: %w", err)
	}
	defer gh.Close()

	return RunGreenhouse(ctx, gh, config)
}

func (cs *ControlSystem) SetInternalInitial(historySize int) {
	cs.heaterHistory = make([]float64, historySize)
	cs.foggerHistory = make([]float64, historySize)
	cs.fanHistory = make([]float64, historySize)
	cs.tempHistory = make([]float64, historySize)
	cs.humidityHistory = make([]float64, historySize)
	cs.historyIndex = 0
	cs.historySize = historySize
}

// RunGreenhouse runs the greenhouse simulation or real environment with the given environment and configuration.
func RunGreenhouse(ctx context.Context, env GreenhouseEnvironment, cfg SimConfig) error {
	fmt.Printf("Starting greenhouse simulation\n")
	if cfg.Duration == 0 {
		fmt.Printf("  Duration: perpetual (press Ctrl+C to stop)\n")
	} else {
		fmt.Printf("  Duration: %v\n", cfg.Duration)
	}
	_ = rand.New(rand.NewSource(cfg.RngSeed))
	logger := NewLogger(cfg.LogFile)
	defer logger.Close()

	var clock Clock
	if cfg.UseMockTime {
		clock = NewMockClock(time.Now())
	} else {
		clock = &RealClock{}
	}

	lrs := LearningRates{
		Weights: learningRateWeights,
		Centers: learningRateCenters,
		Spreads: learningRateSpreads,
	}
	rbfNetwork := NewRecurrentRBFNetwork(initialHiddenNodes, maxHiddenNodes, lrs, 0.05, 0.1)
	rbfNetwork.SetGrowthEnabled(cfg.EnableGrowth)
	rbfNetwork.SetPruningEnabled(cfg.EnablePruning)

	pidController := NewAdaptivePIDController()   // TODO: NewMultiPIDController()
	pidControllerRH := NewAdaptivePIDController() // For humidity control

	// Initialize history buffers (300 samples = 5 minutes @ 1Hz)
	historySize := 300

	system := &ControlSystem{
		Environment: env,
		RbfNetwork:  rbfNetwork,
		PidCtrl:     pidController,
		PidCtrlRH:   pidControllerRH,
		PidOverrider: PidOverrider{
			TBound: FloatBound{
				Min: 50, // TODO: ok?
				Max: 85, // TODO: ok?
			},
			MaxTempDelta: 10.0, // TODO: ok?
		},
		Logger:          logger,
		Clock:           clock,
		Setpoint:        cfg.Setpoint,
		SetpointRH:      65.0, // Default humidity setpoint (%)
		Verbose:         cfg.Verbose,
		TargetTemp:      cfg.Setpoint,
		ErrorHistory:    make([]float64, 0, numErrorHistory),
		SetpointChanges: make(map[time.Duration]float64),
		heaterHistory:   make([]float64, historySize),
		foggerHistory:   make([]float64, historySize),
		fanHistory:      make([]float64, historySize),
		tempHistory:     make([]float64, historySize),
		humidityHistory: make([]float64, historySize),
		historyIndex:    0,
		historySize:     historySize,
	}

	fmt.Printf("  Target setpoint: %.1f°F\n", cfg.Setpoint)
	//fmt.Printf("  Heater delay: %v\n", cfg.HeaterDelay)
	fmt.Printf("  Output file: %s\n", cfg.LogFile)
	fmt.Println()

	startTime := time.Now()
	err := system.Run(ctx, cfg.Duration)
	if err != nil {
		return fmt.Errorf("simulation failed: %w", err)
	}

	elapsed := time.Since(startTime)
	fmt.Printf("\nSimulation completed in %.2f seconds\n", elapsed.Seconds())
	fmt.Printf("Results written to %s\n", cfg.LogFile)

	stats := system.ComputeStatistics()
	fmt.Printf("\n=== Performance Metrics ===\n")
	fmt.Printf("Mean absolute error: %.3f°F\n", stats.MeanAbsError)
	fmt.Printf("RMS error: %.3f°F\n", stats.RMSError)
	fmt.Printf("Max absolute error: %.3f°F\n", stats.MaxAbsError)
	fmt.Printf("Steady-state error (final 5min): %.3f°F\n", stats.SteadyStateError)
	fmt.Printf("Network grew to: %d hidden Nodes (Max %d)\n", stats.FinalNodeCount, system.RbfNetwork.MaxHidden)
	fmt.Printf("Total loss accumulated: %.3f\n", stats.TotalLoss)

	finalRBFPath := "finalRRBFNN.json"
	finalPIDPath := "finalPID.json"

	f, err := os.Create(finalRBFPath)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", finalRBFPath, err)
	}
	defer f.Close()
	if err = json.NewEncoder(f).Encode(system.RbfNetwork); err != nil {
		return fmt.Errorf("failed to write %s: %w", finalRBFPath, err)
	}

	fp, err := os.Create(finalPIDPath)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", finalPIDPath, err)
	}
	defer fp.Close()
	if err = json.NewEncoder(fp).Encode(system.PidCtrl.ShowGains()); err != nil {
		return fmt.Errorf("failed to write %s: %w", finalPIDPath, err)
	}
	return nil
}

// RunGreenhouseSimulation is the main simulation function that can be used as a package export.
// It runs a complete greenhouse control simulation with the specified parameters.
// If duration is 0, runs indefinitely until context is cancelled.
func RunGreenhouseSimulation(ctx context.Context, cfg SimConfig) error {
	rnd := rand.New(rand.NewSource(cfg.RngSeed))
	simulator := NewGreenhouseSimulator(
		initialInternalTemp,
		initialExternalTemp,
		initialInternalHumidity,
		initialExternalHumidity,
		60*time.Second, // TODO: ok?
		rnd,
	)
	defer simulator.Close()
	return RunGreenhouse(ctx, simulator, cfg)
}
