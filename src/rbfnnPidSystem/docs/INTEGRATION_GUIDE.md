# Integration Guide: Using RealGreenhouseEnvironment with ControlSystem

**Date**: October 9, 2026  
**Question**: Can `NewRealGreenhouseEnvironment` be used in `ControlSystem`? Should it be?

**Answer**: ✅ **YES - It should be able to be used in ControlSystem**

Currently, it **cannot** be used directly because `ControlSystem` is tightly coupled to `GreenhouseSimulator`. However, this is a design limitation that should be fixed. Here's why and how.

---

## The Problem

### Current Architecture (Tightly Coupled)

```
ControlSystem
    └─ simulator: *GreenhouseSimulator  ← Only accepts this type
         ├─ GetState() → GreenhouseState
         └─ Step(heaterPower float64) → GreenhouseState
```

### Current Issue

In `control_system.go`, the `step()` method uses:
```go
state := cs.simulator.GetState()      // Line 85
_ = cs.simulator.Step(pidOut.HeaterPower)  // Line 106
```

`RealGreenhouseEnvironment` has **different methods**:
- `ReadSensors() (GreenhouseInput, error)`
- `SendControl(ControlOutput) error`
- `GetCurrentHeaterPower() float64`

**They are incompatible interfaces!**

---

## The Solution

Create a common interface that both can implement:

### Step 1: Define the Interface

Add this to `types.go`:

```go
// GreenhouseEnvironment is a common interface for both simulated and real greenhouses
type GreenhouseEnvironment interface {
	// ReadSensors returns current greenhouse state
	ReadSensors() (GreenhouseState, error)
	
	// SendControl sends control command to heater
	SendControl(power float64) error
	
	// GetCurrentHeaterPower returns the heater power level
	GetCurrentHeaterPower() float64
	
	// Close cleans up resources
	Close() error
}
```

### Step 2: Adapt GreenhouseSimulator

Add these methods to `simulator.go`:

```go
// ReadSensors returns the current state as GreenhouseState
func (g *GreenhouseSimulator) ReadSensors() (GreenhouseState, error) {
	return g.GetState(), nil
}

// SendControl accepts a heater power value
func (g *GreenhouseSimulator) SendControl(power float64) error {
	g.Step(power)
	return nil
}

// GetCurrentHeaterPower returns current heater power
func (g *GreenhouseSimulator) GetCurrentHeaterPower() float64 {
	return g.GetState().HeaterPowerOutput
}

// Close is a no-op for simulator
func (g *GreenhouseSimulator) Close() error {
	return nil
}
```

### Step 3: Adapt RealGreenhouseEnvironment

The `RealGreenhouseEnvironment` already has most of what we need. Add this adapter:

```go
// GreenhouseInput represents sensor reading (used by RealGreenhouseEnvironment)
type GreenhouseInput struct {
	InternalTemp     float64
	InternalHumidity float64
	ExternalTemp     float64
	ExternalHumidity float64
	Timestamp        time.Time
}

// Adapt RealGreenhouseEnvironment to GreenhouseEnvironment interface
// It already has ReadSensors() and SendControl(), but we need to bridge formats

// Add helper method to convert GreenhouseInput to GreenhouseState
func (r *RealGreenhouseEnvironment) GetState() (GreenhouseState, error) {
	input, err := r.ReadSensors()
	if err != nil {
		return GreenhouseState{}, err
	}
	
	return GreenhouseState{
		Time:              input.Timestamp,
		InternalTemp:      input.InternalTemp,
		InternalHumidity:  input.InternalHumidity,
		ExternalTemp:      input.ExternalTemp,
		ExternalHumidity:  input.ExternalHumidity,
		HeaterPowerOutput: r.GetCurrentHeaterPower(),
	}, nil
}

// Override SendControl to accept GreenhouseState-compatible format
// Already implemented as SendControl(output ControlOutput)
```

### Step 4: Refactor ControlSystem

Replace the tightly-coupled simulator with the interface:

**In `types.go`:**
```go
// BEFORE:
type ControlSystem struct {
	simulator       *GreenhouseSimulator  ← Concrete type
	...
}

// AFTER:
type ControlSystem struct {
	environment     GreenhouseEnvironment  ← Interface (accepts both)
	...
}
```

**In `control_system.go` step() method:**
```go
// BEFORE:
state := cs.simulator.GetState()
_ = cs.simulator.Step(pidOut.HeaterPower)

// AFTER:
state, err := cs.environment.ReadSensors()
if err != nil {
	return fmt.Errorf("sensor read error: %w", err)
}

err = cs.environment.SendControl(pidOut.HeaterPower)
if err != nil {
	return fmt.Errorf("control error: %w", err)
}
```

### Step 5: Update Main to Support Both

```go
func RunGreenhouseSimulation(ctx context.Context, cfg SimConfig) error {
	// ... existing setup code ...
	
	// Create simulator (same as before)
	simulator := NewGreenhouseSimulator(...)
	var env GreenhouseEnvironment = simulator  // Use interface
	
	// OR use real hardware:
	// sensors := NewDHTSensors()
	// actuators := NewPWMHeater("GPIO18")
	// realEnv, _ := NewRealGreenhouseEnvironment(sensors, actuators, ...)
	// var env GreenhouseEnvironment = realEnv
	
	// Create ControlSystem with interface
	system := &ControlSystem{
		environment:     env,  // Now accepts both!
		rbfNetwork:      rbfNetwork,
		pidCtrl:         pidController,
		...
	}
	
	// Run exactly the same - no changes needed!
	return system.Run(ctx, cfg.simDuration)
}
```

---

## Benefits of This Approach

### 1. **Dependency Inversion**
```
Before: ControlSystem → GreenhouseSimulator (concrete)
After:  ControlSystem → GreenhouseEnvironment (interface)
```

### 2. **Swap Interchangeably**
```go
// Same code, different behavior
env1 := NewGreenhouseSimulator(...)          // Simulation
env2, _ := NewRealGreenhouseEnvironment(...) // Real hardware

system := &ControlSystem{environment: env1}  // Either works!
// or
system := &ControlSystem{environment: env2}  // Either works!
```

### 3. **Easy Testing**
```go
// Mock implementation for testing
type MockEnvironment struct {
	tempReadings []float64
}

func (m *MockEnvironment) ReadSensors() (GreenhouseState, error) {
	// Return predefined values
}

func (m *MockEnvironment) SendControl(power float64) error {
	// Verify control commands
}

func (m *MockEnvironment) GetCurrentHeaterPower() float64 {
	return 0.5
}

// Use in tests:
system := &ControlSystem{environment: &MockEnvironment{}}
```

### 4. **Future Extensions**
```go
// Easy to add new implementations:
// - NetworkGreenhouseEnvironment (controls remote greenhouse)
// - HybridGreenhouseEnvironment (combines sim + real)
// - RecordingGreenhouseEnvironment (records and replays)
```

---

## Complete Code Example

### File: types.go (additions)

```go
// GreenhouseEnvironment is a common interface for all greenhouse implementations
type GreenhouseEnvironment interface {
	// ReadSensors returns the current greenhouse state
	// Returns error if sensors are unavailable or outdated
	ReadSensors() (GreenhouseState, error)
	
	// SendControl sends a control command to the actuators
	// power should be in range [0, 1]
	SendControl(power float64) error
	
	// GetCurrentHeaterPower returns the last commanded heater power
	GetCurrentHeaterPower() float64
	
	// Close performs cleanup (close connections, turn off actuators, etc.)
	Close() error
}

// GreenhouseInput represents raw sensor data (used by real environments)
type GreenhouseInput struct {
	InternalTemp     float64
	InternalHumidity float64
	ExternalTemp     float64
	ExternalHumidity float64
	Timestamp        time.Time
}

// ControlOutput represents actuator commands (used by real environments)
type ControlOutput struct {
	HeaterPower float64
}
```

### File: control_system.go (refactored step method)

```go
func (cs *ControlSystem) step() error {
	// Read from environment (works with both simulator and real hardware)
	state, err := cs.environment.ReadSensors()
	if err != nil {
		// Handle sensor error gracefully
		if cs.verbose {
			fmt.Printf("⚠ Sensor error: %v\n", err)
		}
		return err
	}
	
	// Compute error
	error := state.InternalTemp - cs.setpoint
	cs.errorHistory = append(cs.errorHistory, error)
	
	// Get RBF network output
	inputs := cs.constructNetworkInputs(state, error)
	if len(inputs) != 41 {
		return fmt.Errorf("inputs has %d dimensions, expected 41", len(inputs))
	}
	
	rbfOut := cs.rbfNetwork.Forward(inputs)
	
	// Compute control action
	dt := 1 * time.Second
	pidOut := cs.pidCtrl.Compute(error, rbfOut.Kp, rbfOut.Ki, rbfOut.Kd, dt)
	
	// Send control to environment (works with both)
	if err := cs.environment.SendControl(pidOut.HeaterPower); err != nil {
		if cs.verbose {
			fmt.Printf("⚠ Control error: %v\n", err)
		}
		return err
	}
	
	// Online learning
	cs.rbfNetwork.BackwardAndUpdate(error, rbfOut.HiddenActivations, rbfOut.AugmentedInputs, rbfOut.PreSigmoidRaw)
	cs.rbfNetwork.AdaptStructure(inputs, error)
	
	cs.totalLoss += error * error
	
	// Logging
	if cs.verbose && len(cs.errorHistory)%100 == 0 {
		fmt.Printf("Step %d: Temp=%.1f°F, Setpoint=%.1f°F, Error=%.2f°F, Heater=%.3f, Nodes=%d\n",
			len(cs.errorHistory),
			state.InternalTemp,
			cs.setpoint,
			error,
			pidOut.HeaterPower,
			cs.rbfNetwork.GetNumNodes(),
		)
	}
	
	return nil
}
```

### File: types.go (ControlSystem update)

```go
type ControlSystem struct {
	environment     GreenhouseEnvironment  // Changed from *GreelhouseSimulator
	rbfNetwork      *RecurrentRBFNetwork
	pidCtrl         *AdaptivePIDController
	logger          *Logger
	clock           Clock
	setpoint        float64
	targetTemp      float64
	verbose         bool
	errorHistory    []float64
	totalLoss       float64
	lastLogged      time.Time
	setpointChanges map[time.Duration]float64
}
```

---

## Migration Path

### Phase 1: Add Interface (No Breaking Changes)
1. Define `GreenhouseEnvironment` interface
2. Keep `ControlSystem.simulator` as is
3. Add adapters to both simulator and real environment

### Phase 2: Refactor ControlSystem (Optional Breaking Change)
1. Replace `simulator` field with `environment`
2. Update `step()` method
3. Maintain backward compatibility via factory function

### Phase 3: Update CLI and Examples
1. Show both simulator and real hardware usage
2. Document the interface
3. Provide example implementations

---

## Usage After Refactoring

### Using Simulator (same as before)
```go
func main() {
	ctx := context.Background()
	
	simulator := NewGreenhouseSimulator(
		70.0, 60.0, 75.0, 50.0,
		90*time.Second,
	)
	
	rbfNetwork := NewRecurrentRBFNetwork(10, 50, lrs)
	pidCtrl := NewAdaptivePIDController()
	
	system := &ControlSystem{
		environment: simulator,  // Interface accepts concrete type
		rbfNetwork: rbfNetwork,
		pidCtrl: pidCtrl,
		...
	}
	
	system.Run(ctx, 5*time.Minute)
}
```

### Using Real Hardware (new capability)
```go
func main() {
	ctx := context.Background()
	
	// Create hardware
	sensors := NewDHTSensors()
	actuators := NewPWMHeater("GPIO18")
	
	// Create real environment
	realEnv, _ := NewRealGreenhouseEnvironment(
		sensors, actuators,
		500*time.Millisecond,
		500*time.Millisecond,
	)
	
	// Use exactly the same ControlSystem code!
	system := &ControlSystem{
		environment: realEnv,  // Same interface
		rbfNetwork: rbfNetwork,
		pidCtrl: pidCtrl,
		...
	}
	
	system.Run(ctx, 24*time.Hour)
}
```

---

## Summary

**Current State**: `RealGreenhouseEnvironment` **cannot** be used with `ControlSystem`
- ✗ Incompatible interfaces
- ✗ Tightly coupled to `GreenhouseSimulator`
- ✗ Different method signatures

**Solution**: Create `GreenhouseEnvironment` interface
- ✓ Define common interface
- ✓ Adapt both implementations
- ✓ Refactor `ControlSystem` to use interface
- ✓ Same code works with simulator OR real hardware

**Benefits**:
- ✓ Easy to swap between simulation and real hardware
- ✓ Easy to add new implementations (mock, recording, hybrid)
- ✓ Better testability
- ✓ Follows dependency inversion principle

**Recommendation**: YES - This refactoring should be done for production use!

---

## Implementation Checklist

- [ ] Define `GreenhouseEnvironment` interface in `types.go`
- [ ] Define `GreenhouseInput` and `ControlOutput` types
- [ ] Add adapter methods to `GreenhouseSimulator`
- [ ] Verify `RealGreenhouseEnvironment` conforms to interface
- [ ] Refactor `ControlSystem.step()` to use interface
- [ ] Update `types.go` ControlSystem definition
- [ ] Test with simulator (backward compatibility)
- [ ] Test with real hardware
- [ ] Update documentation and examples
- [ ] Verify all existing tests still pass

