package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"try3/rbfnnPidSystem"
)

const (
	//learningRateWeights     = 0.015
	//learningRateCenters     = 0.008
	//learningRateSpreads     = 0.003
	//initialHiddenNodes      = 10
	//maxHiddenNodes          = 50
	//numErrorHistory         = 1000
	//initialInternalTemp     = 70.0
	//initialExternalTemp     = 60.0
	//initialInternalHumidity = 75.0
	//initialExternalHumidity = 50.0
	initialDefaultSetpointF = 72.0
)

func main() {
	duration := flag.Duration("duration", 0, "Duration of simulation (0 = run perpetually, must use Ctrl+C to stop)")
	setpoint := flag.Float64("setpoint", initialDefaultSetpointF, "Target greenhouse temperature (F)")
	logFile := flag.String("log", "greenhouse_sim.csv", "Output CSV log file")
	verbose := flag.Bool("verbose", false, "Print detailed output")
	mockTime := flag.Bool("mock-time", false, "Use mock time for fast testing")
	dynamic := flag.Bool("dynamic", false, "Run dynamic setpoint change experiment")
	seed := flag.Int64("seed", 42, "Random seed")
	enableGrowth := flag.Bool("grow", true, "Enable network node growth")
	enablePruning := flag.Bool("prune", false, "Enable network node pruning")

	flag.Parse()

	if *dynamic {
		mainDynamic(*duration, *setpoint, *seed)
		return
	}

	ctx := context.Background()
	cfg := coreSystem.SimConfig{
		Duration:      *duration,
		Setpoint:      *setpoint,
		LogFile:       *logFile,
		Verbose:       *verbose,
		UseMockTime:   *mockTime,
		EnableGrowth:  *enableGrowth,
		EnablePruning: *enablePruning,
		RngSeed:       *seed,
	}

	if *duration == 0 {
		fmt.Println("Running in perpetual mode (no duration limit)")
		fmt.Println("Press Ctrl+C to stop the simulation")
		fmt.Println()
	}

	err := coreSystem.RunGreenhouseSimulation(ctx, cfg)
	if err != nil {
		log.Fatalf("Simulation failed: %v", err)
	}
}
