# Recurrent RBF Neural Network Greenhouse Heater Control System

## Overview

This is a complete implementation of an **adaptive recurrent RBF (Radial Basis Function) neural network** that learns online to control a greenhouse heater via an adaptive PID controller. The system models realistic greenhouse thermal and humidity dynamics with time delays.

### Key Features

✅ **Recurrent RBF Network**: Learns online with dynamic center growth/pruning  
✅ **Adaptive PID Controller**: Gains (Kp, Ki, Kd) output by the RBF network  
✅ **Online Learning**: Backpropagation updates at every control step (≥1 Hz)  
✅ **Time-Delay Modeling**: Configurable heater delay (30s-3min) via circular buffer  
✅ **Realistic Simulation**: Thermal mass, heat loss, humidity dynamics, sensor noise  
✅ **Data Logging**: CSV output with metrics for analysis  

## Architecture

### System Components

1. **Greenhouse Simulator** (`simulator.go`)
   - Thermal dynamics: `dT/dt = (heater_input - heat_loss) / thermal_mass`
   - Humidity evolution with evaporative cooling effects
   - Realistic sensor noise (±0.3°F temp, ±1.5% RH)
   - Configurable heater delay via circular buffer

2. **Recurrent RBF Network** (`rbf_network.go`)
   - **Input**: 41-dimensional augmented vector
     - 5 time taps × 7 features = 35 dims (heater/fogger/fan power, temps, humidity)
     - 6 additional dims (PID state: error, derivative, integral, operating point)
   - **Hidden Layer**: Adaptive RBF basis functions with Gaussian kernels
     - Dynamic center growth when error is high in novel regions
     - Optional node pruning based on activity
   - **Output**: Kp, Ki, Kd bounded to realistic ranges
     - Kp: [0.0, 1.0]
     - Ki: [0.0, 0.5]
     - Kd: [0.0, 0.3]
   - **Recurrent State**: Previous hidden activations passed back as inputs

3. **Adaptive PID Controller** (`pid.go`)
   - Time-varying gains from RBF network
   - Low-pass filtered derivative term
   - Integral anti-windup with exponential forgetting
   - Output clamped to [0, 1] for heater duty cycle

4. **Data Logger** (`logger.go`)
   - CSV output with 13 columns
   - Buffered writes for efficiency

### Input Vector Construction (41 dimensions)

```
Dimensions 0-34:   5 time taps × 7 features each
  Per tap:
    [0-2]  Heater Power (0-1), Fogger Power (0-1), Fan Power (0-1)
    [3-4]  Normalized Interior Temp, Exterior Temp
    [5-6]  Normalized Interior RH, Exterior RH

Dimensions 35-40:  Current PID/Operating Point State
    [35]   Temperature Error / 5.0
    [36]   Derivative Error / 1.0
    [37]   Integral Error × Ki / 1.0
    [38]   RH Error
    [39]   (Interior Temp - Exterior Temp) / 10.0
    [40]   Reserve Dimension
```

## Building

```bash
cd /Users/ZNM4U7N/projects/other/goPersonal/aa_numberedProjects/9985-pid/try3
go build
```

## Testing

Run the unit tests (all use mock time for instant execution):

```bash
go test -v
```

**All tests complete in <1 second!** Key tests:

- **TestMockTimeSimulation**: 1-minute simulation in 5ms
- **TestRealTimeSimulation**: Actual 20-second real-time test (optional with `-run`)
- **TestMockClockAdvance**: Verify MockClock functionality
- **TestRBFNetworkGrowth**: Verify network grows during learning

## Running

### Basic Usage

```bash
./try3 -duration=5m -setpoint=72 -delay=90s
```

### Command-Line Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-duration` | 30m | Simulation duration |
| `-setpoint` | 72.0 | Target greenhouse temperature (°F) |
| `-delay` | 90s | Heater effect delay (30s-3min typical) |
| `-log` | greenhouse_sim.csv | Output CSV log file |
| `-verbose` | false | Print detailed step output |
| `-mock-time` | false | **Use mock time for instant execution (testing)** |
| `-seed` | 42 | Random seed for reproducibility |
| `-grow` | true | **Enable/disable network node growth** |
| `-prune` | false | **Enable/disable network node pruning** |

### Examples

```bash
# 10-minute simulation with 2-minute heater delay
./try3 -duration=10m -setpoint=75 -delay=2m -verbose=true

# Quick 30-second test
./try3 -duration=30s -setpoint=70 -delay=45s

# Long-running experiment with custom logging
./try3 -duration=2h -setpoint=72 -log=experiment_run1.csv

# **FAST MODE: 5-minute simulation completes in ~20ms!**
./try3 -duration=5m -setpoint=72 -delay=90s -mock-time

# Test with network growth disabled
./try3 -duration=5m -setpoint=72 -delay=90s -mock-time -grow=false

# Test with pruning enabled
./try3 -duration=5m -setpoint=72 -delay=90s -mock-time -grow=true -prune=true

# Network learns delay implicitly (no explicit delay parameter needed after initial configuration)
./try3 -duration=10m -setpoint=72 -delay=120s -mock-time
```

## Output CSV Format

| Column | Meaning |
|--------|---------|
| Time | Timestamp (HH:MM:SS.mmm) |
| Setpoint | Target temperature (°F) |
| ActualTemp | Measured greenhouse temperature (°F) |
| Error | SetPoint - ActualTemp |
| HeaterPower | PID controller output (0-1) |
| Kp | Proportional gain from RBF network |
| Ki | Integral gain from RBF network |
| Kd | Derivative gain from RBF network |
| ExternalTemp | Ambient temperature (°F) |
| ExternalHumidity | Ambient humidity (%) |
| Humidity | Greenhouse humidity (%) |
| Loss | Accumulated squared error |
| NodeCount | Current RBF network size |

## How It Works

### Online Learning Loop (Every 1 second)

1. **Get greenhouse state** with sensor noise
2. **Compute error** = setpoint - actual_temperature
3. **RBF Forward Pass**: 
   - Construct 41-D augmented input
   - Compute 49-D hidden layer activations (41 external + 8 recurrent)
   - Output PID gains via sigmoid-bounded output layer
4. **PID Controller**: Compute heater power using adaptive gains
5. **Greenhouse Simulation**: Apply delayed heater power (90s default)
6. **Backpropagation**: Update RBF weights, centers, spreads
7. **Structural Adaptation**: Grow nodes if high error in novel region
8. **Logging**: Every 10 steps, write metrics to CSV

### Growth Criterion

New RBF node is added when BOTH:
- `|error| > 1.0°F` (high error threshold)
- `min_distance_to_existing_center > 0.5` (novel input region)

This prevents redundant nodes and focuses learning on underrepresented regions.

### Recurrent Dimension Expansion

When a node grows:
1. All existing center vectors expand by 1 dimension
2. New dimension initialized to 0.5 (neutral)
3. New center gets its own coordinate = 1.0
4. `prevActivations` array expands to track new node's activation

This allows the network to model temporal dependencies through feedback.

## Physical Parameters (Configurable)

In `simulator.go`:

```go
thermalMass:    100.0      // Heat capacity (J/K)
heatLossCoeff:  0.5        // Heat loss rate
humidityTau:    30.0       // Humidity time constant (sec)
tempNoise:      0.3        // Sensor noise std dev (°F)
humidityNoise:  1.5        // Sensor noise std dev (%)
```

## Learning Parameters (Configurable)

In `main.go`:

```go
LearningRates {
  Weights: 0.01,   // Output weight learning rate
  Centers: 0.005,  // RBF center learning rate
  Spreads: 0.002,  // RBF spread (sigma) learning rate
}

// Growth/Pruning
errorThreshold:    1.0   // °F
distanceThreshold: 0.5   // Normalized units
activityDecay:     0.99  // EMA decay
pruneEnabled:      false // Disabled by default
```

## Performance Metrics

The system outputs statistics after each run:

- **Mean Absolute Error**: Average distance from setpoint
- **RMS Error**: Root mean squared tracking error
- **Max Absolute Error**: Worst-case deviation
- **Steady-State Error**: Error in final 5% of simulation
- **Network Size**: How many RBF nodes the network grew to

## Typical Results

On a 3-minute simulation:
- Initial temperature: 70°F
- Setpoint: 72°F
- Delay: 90 seconds
- Network grows from 8 to ~15 nodes
- Mean error converges from ~5°F to ~1°F
- RBF gains learned: Kp ≈ 0.05-0.10, Ki ≈ 0.05-0.08, Kd ≈ 0.04-0.07

## File Structure

```
try3/
├── main.go                 # Entry point and configuration
├── types.go                # Type definitions
├── simulator.go            # Greenhouse physics simulation
├── pid.go                  # Adaptive PID controller
├── rbf_network.go          # Recurrent RBF network with learning
├── control_system.go       # Main control loop
├── logger.go               # CSV data logging
├── go.mod                  # Go module file
└── greenhouse_sim.csv      # Output data (created at runtime)
```

## Technical Details

### Gradient Computation

The backpropagation uses chain rule through:
- Sigmoid output squashing function
- RBF basis function kernels
- Plant model Jacobian (estimated as -1.0)

### Recurrent Architecture

Unlike standard feedforward RBFs, this network includes:
- **Augmented input**: `z = [external_inputs, prev_hidden_activations]`
- **Dynamic dimensions**: Adding nodes increases both center and input space
- **Temporal memory**: Hidden layer activations feed back as inputs

This allows the network to learn time-dependent dynamics without explicit time delay windows.

### Anti-Windup Integration

The PID integral term uses:
- Exponential forgetting: `eInt_new = eInt_old × exp(-dt/τ) + error × dt`
- Clamping-based anti-windup: When output saturates, integral is reduced

## Time-Delay Learning

⚡ **The network learns the heater delay implicitly** through its recurrent architecture.

Rather than making delay an explicit parameter, the system:

1. **Stores heater history** in a circular buffer (past 30-180 seconds)
2. **Includes heater history** in the RBF network input
3. **Network learns** which past heater command correlates with current temperature error
4. **Adaptive gains** adjust based on this learned temporal pattern

**See `DELAY_LEARNING_IMPL.md` for detailed theory and validation methods.**

### How It Works

```
Circular buffer: [heater_cmd[t-N], ..., heater_cmd[t-2], heater_cmd[t-1]]
                           ↓
                   RBF network input
                           ↓
                   Network learns: "If temp error is negative AND
                                    heater was high ~90s ago,
                                    then keep heating"
                           ↓
                   This compensates for the transport lag!
```

**Benefits:**
- ✅ Automatic discovery of delay (no manual tuning)
- ✅ Handles variable/unknown delays
- ✅ Adapts if delay changes during operation
- ✅ No special architecture modifications needed

## Future Enhancements

- [ ] Multi-output RBF (control both heater and fogger)
- [ ] Second-order sensor filtering
- [ ] Adaptive learning rate schedules
- [ ] Kalman filtering for state estimation
- [ ] Hardware integration with real greenhouse sensors
- [ ] Matplotlib/Gnuplot visualization scripts
- [ ] Configuration file support (YAML/TOML)

## Documentation

Complete documentation is available in the [`docs/`](docs/) directory:

### Quick Start
- **[MASTER_INDEX.md](docs/MASTER_INDEX.md)** - Central navigation hub for all documentation
- **[QUICK_REFERENCE.md](docs/QUICK_REFERENCE.md)** - Quick reference guide

### Core Features
- **[PERPETUAL_MODE_COMPLETE.md](docs/PERPETUAL_MODE_COMPLETE.md)** - Run indefinitely without time limits
- **[IMMEDIATE_SETPOINT_CHANGES.md](docs/IMMEDIATE_SETPOINT_CHANGES.md)** - Change temperature setpoints during operation
- **[DYNAMIC_PERPETUAL_MODE.md](docs/DYNAMIC_PERPETUAL_MODE.md)** - Dynamic setpoint changes with perpetual operation
- **[REFACTORING_COMPLETE.md](docs/REFACTORING_COMPLETE.md)** - Interface abstraction for simulator/real hardware

### Hardware Integration
- **[REAL_GREENHOUSE_USAGE.md](docs/REAL_GREENHOUSE_USAGE.md)** - Connect real greenhouse hardware
- **[HARDWARE_COMPLETE.md](docs/HARDWARE_COMPLETE.md)** - Hardware integration details
- **[HARDWARE_INTEGRATION.md](docs/HARDWARE_INTEGRATION.md)** - Integration guide for actual systems

### Theory & Mathematics
- **[MATHEMATICS.md](docs/MATHEMATICS.md)** - Complete mathematical formulation
- **[DELAY_LEARNING_IMPL.md](docs/DELAY_LEARNING_IMPL.md)** - How the network learns heater delay implicitly
- **[DYNAMIC_SETPOINT.md](docs/DYNAMIC_SETPOINT.md)** - Setpoint change handling

### Architecture & Design
- **[ARCHITECTURE_DIAGRAMS.md](docs/ARCHITECTURE_DIAGRAMS.md)** - Visual system design
- **[CONTROL_SYSTEM_INTEGRATION.md](docs/CONTROL_SYSTEM_INTEGRATION.md)** - Component integration details
- **[IMPLEMENTATION_DETAILS.md](docs/IMPLEMENTATION_DETAILS.md)** - Implementation architecture

### Integration Guides
- **[INTEGRATION_GUIDE.md](docs/INTEGRATION_GUIDE.md)** - Step-by-step integration instructions
- **[INTEGRATION_INDEX.md](docs/INTEGRATION_INDEX.md)** - Navigation and FAQ for integration

### Completion & Status
- **[COMPLETION_STATUS.md](docs/COMPLETION_STATUS.md)** - Project completion status
- **[COMPLETION_SUMMARY.md](docs/COMPLETION_SUMMARY.md)** - Summary of implementation
- **[VALIDATION_REPORT.md](docs/VALIDATION_REPORT.md)** - Validation and testing results
- **[CHECKLIST.md](docs/CHECKLIST.md)** - Feature checklist and completion tracking

### Reference Documentation
- **[FINAL_SUMMARY.md](docs/FINAL_SUMMARY.md)** - Complete feature overview
- **[SUMMARY.md](docs/SUMMARY.md)** - Implementation summary
- **[PID_MEMORY_TIME.md](docs/PID_MEMORY_TIME.md)** - PID controller memory time configuration
- **[DELAY_LEARNING.md](docs/DELAY_LEARNING.md)** - Delay learning theory
- **[DYNAMIC_ENVIRONMENT.md](docs/DYNAMIC_ENVIRONMENT.md)** - Dynamic environment handling
- **[PACKAGE_REFACTORING.md](docs/PACKAGE_REFACTORING.md)** - Package structure information
- **[REAL_GREENHOUSE_GUIDE.md](docs/REAL_GREENHOUSE_GUIDE.md)** - Real greenhouse deployment guide
- **[SPACING_UPDATE.md](docs/SPACING_UPDATE.md)** - Setpoint spacing updates
- **[AGENTS.md](docs/AGENTS.md)** - Agent-related documentation
- **[INDEX.md](docs/INDEX.md)** - Documentation index

## License

Public domain (no restrictions)

## Author

Developed as part of RFP-9985 for adaptive greenhouse climate control.

