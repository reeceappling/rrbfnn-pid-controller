# 📚 Master Documentation Index

**Quick Navigation for the Recurrent RBF Greenhouse Control System**

---

## 🎯 Getting Started

### For First-Time Users
1. **[README.md](README.md)** - Overview, features, and building
2. **[FINAL_SUMMARY.md](FINAL_SUMMARY.md)** - Complete feature list and architecture

### For Quick Tests
```bash
./greenhouse_sim -mock-time -duration 5m      # 5-minute test (instant)
./greenhouse_sim -dynamic -mock-time           # Dynamic setpoint test
go test -v                                     # Run all tests
```

---

## 📖 Documentation by Topic

### Core Features & Architecture
- **[REFACTORING_COMPLETE.md](REFACTORING_COMPLETE.md)** - Interface abstraction for simulator/real hardware
- **[PERPETUAL_MODE_COMPLETE.md](PERPETUAL_MODE_COMPLETE.md)** - Run indefinitely without time limits
- **[DYNAMIC_PERPETUAL_MODE.md](DYNAMIC_PERPETUAL_MODE.md)** - Dynamic setpoint changes with perpetual mode
- **[IMMEDIATE_SETPOINT_CHANGES.md](IMMEDIATE_SETPOINT_CHANGES.md)** - On-the-fly setpoint control via API

### Hardware Integration
- **[REAL_GREENHOUSE_USAGE.md](REAL_GREENHOUSE_USAGE.md)** - How to connect real hardware
- **real_greenhouse.go** - Production implementation for actual greenhouses

### Theory & Mathematics
- **[MATHEMATICS.md](MATHEMATICS.md)** - Complete mathematical formulation
- **[DYNAMIC_SETPOINT.md](DYNAMIC_SETPOINT.md)** - Details on setpoint change handling

### Design & Architecture
- **[ARCHITECTURE_DIAGRAMS.md](ARCHITECTURE_DIAGRAMS.md)** - Visual system design
- **[CONTROL_SYSTEM_INTEGRATION.md](CONTROL_SYSTEM_INTEGRATION.md)** - How components integrate
- **[INTEGRATION_GUIDE.md](INTEGRATION_GUIDE.md)** - Step-by-step integration guide
- **[INTEGRATION_INDEX.md](INTEGRATION_INDEX.md)** - Navigation and FAQ

### Code Organization
- **main.go** - CLI interface and exported API
- **control_system.go** - Core control loop (Run(), step())
- **rbf_network.go** - Recurrent RBF network with online learning
- **pid_controller.go** - Adaptive PID controller
- **simulator.go** - Greenhouse physics simulator
- **types.go** - Type definitions and interfaces
- **real_greenhouse.go** - Real hardware integration
- **hardware.go** - Deprecated interfaces (kept for reference)

---

## 🛠️ By Use Case

### I Want To...

#### **Run a quick test**
```bash
./greenhouse_sim -mock-time -duration 1m
```
See: [PERPETUAL_MODE_COMPLETE.md](PERPETUAL_MODE_COMPLETE.md)

#### **Use as a CLI tool**
```bash
./greenhouse_sim -setpoint 75 -duration 1h -log results.csv -verbose
```
See: [README.md](README.md)

#### **Use as an importable package**
```go
import "try3"
cfg := SimConfig{Duration: 5*time.Minute, Setpoint: 72.0, ...}
err := RunGreenhouseSimulation(ctx, cfg)
```
See: [FINAL_SUMMARY.md](FINAL_SUMMARY.md) - Programmatic Usage

#### **Change temperature setpoints on the fly**
```go
system.SetSetpointImmediate(75.0)  // Change anytime
```
See: [IMMEDIATE_SETPOINT_CHANGES.md](IMMEDIATE_SETPOINT_CHANGES.md)

#### **Connect real greenhouse hardware**
```go
sensors := NewMySensors()
actuators := NewMyActuators()
gh, _ := NewRealGreenhouseEnvironment(sensors, actuators, ...)
err := RunGreenhouse(ctx, gh, cfg)
```
See: [REAL_GREENHOUSE_USAGE.md](REAL_GREENHOUSE_USAGE.md)

#### **Test with dynamic setpoint changes**
```bash
./greenhouse_sim -dynamic
```
See: [DYNAMIC_PERPETUAL_MODE.md](DYNAMIC_PERPETUAL_MODE.md)

#### **Understand the math**
See: [MATHEMATICS.md](MATHEMATICS.md)

#### **Deploy to production**
See: [FINAL_SUMMARY.md](FINAL_SUMMARY.md) - Production Ready section

---

## 📊 Features Checklist

### ✅ Core
- Recurrent RBF network with online learning
- Adaptive PID controller
- Time-delay modeling
- Online backpropagation
- Dynamic setpoint changes

### ✅ Flexibility
- Interface-based design (works with simulator, real hardware, mocks)
- Perpetual (indefinite) operation mode
- Mock time for fast testing
- Network growth/pruning control

### ✅ Real World
- Anti-windup protection
- Sensor error handling
- Graceful shutdown
- Thread-safe API calls
- Logging for audit trail

---

## 🧪 Testing

### Run All Tests
```bash
go test -v
```

### Run Specific Scenarios
```bash
# Dynamic setpoint changes
./greenhouse_sim -dynamic -mock-time

# Long-term test with perpetual mode
./greenhouse_sim -duration 0

# High verbosity
./greenhouse_sim -verbose -duration 5m -mock-time
```

### Expected Results
- All 6 tests should pass in ~20 seconds
- Errors should converge to within ±2°F steady-state
- Network should grow from 8→10-15 nodes

---

## 📋 Reference

### Command-Line Flags
```
-duration     Duration of simulation (0 = perpetual)
-setpoint     Target temperature (°F)
-delay        Heater response delay
-log          Output CSV filename
-verbose      Enable detailed output
-mock-time    Use fast mock clock
-dynamic      Run dynamic setpoint test
-seed         Random seed for reproducibility
-grow         Enable/disable network growth
-prune        Enable/disable network pruning
```

### Key Types
- `GreenhouseEnvironment` - Interface for any greenhouse (simulator, real, mock)
- `ControlSystem` - Main control loop, call `Run(ctx, duration)`
- `RecurrentRBFNetwork` - Neural network with online learning
- `AdaptivePIDController` - PID with adaptive gains from network
- `GreenhouseSimulator` - Physics simulation
- `RealGreenhouseEnvironment` - Production hardware integration

### Key Methods
- `RunGreenhouseSimulation(ctx, cfg)` - Simple way to run simulator
- `RunGreenhouse(ctx, env, cfg)` - Generic runner for any environment
- `system.SetSetpointImmediate(temp)` - Change setpoint during operation
- `system.GetSetpoint()` - Read current setpoint
- `system.ChangeSetpoint(time, temp)` - Pre-schedule changes
- `system.ComputeStatistics()` - Get performance metrics

---

## 🚀 Quick Start Paths

### Developer Path
1. Read: [FINAL_SUMMARY.md](FINAL_SUMMARY.md)
2. Build: `go build -o greenhouse_sim`
3. Test: `./greenhouse_sim -mock-time -duration 1m`
4. Explore: `go test -v`

### Researcher Path
1. Read: [MATHEMATICS.md](MATHEMATICS.md)
2. Read: [ARCHITECTURE_DIAGRAMS.md](ARCHITECTURE_DIAGRAMS.md)
3. Modify: Edit learning rates in constants
4. Test: `go test -v`

### Hardware Integration Path
1. Read: [REAL_GREENHOUSE_USAGE.md](REAL_GREENHOUSE_USAGE.md)
2. Implement: `HardwareSensors` and `HardwareActuators` interfaces
3. Test: Create with `NewRealGreenhouseEnvironment()`
4. Deploy: Use `RunGreenhouse()` with your environment

### Production Path
1. Read: [FINAL_SUMMARY.md](FINAL_SUMMARY.md) - Production section
2. Review: [REAL_GREENHOUSE_USAGE.md](REAL_GREENHOUSE_USAGE.md)
3. Implement: Your sensor/actuator interfaces
4. Deploy: Use `RunGreenhouse()` or `RunGreenhouseSimulation()`

---

## ❓ FAQ

**Q: How do I change temperature during operation?**
A: Use `system.SetSetpointImmediate(newTemp)` - See [IMMEDIATE_SETPOINT_CHANGES.md](IMMEDIATE_SETPOINT_CHANGES.md)

**Q: Can I run without a time limit?**
A: Yes, use `-duration 0` - See [PERPETUAL_MODE_COMPLETE.md](PERPETUAL_MODE_COMPLETE.md)

**Q: How do I connect real hardware?**
A: Implement interfaces and use `RunGreenhouse()` - See [REAL_GREENHOUSE_USAGE.md](REAL_GREENHOUSE_USAGE.md)

**Q: How fast does it run?**
A: 1-5 minute simulations complete in <20ms with mock time - See [PERPETUAL_MODE_COMPLETE.md](PERPETUAL_MODE_COMPLETE.md)

**Q: Can I use this as a library?**
A: Yes, export function `RunGreenhouseSimulation(ctx, cfg)` - See [FINAL_SUMMARY.md](FINAL_SUMMARY.md)

**Q: What's the thread-safety model?**
A: Main loop is single-threaded. `SetSetpointImmediate()` is thread-safe. - See [IMMEDIATE_SETPOINT_CHANGES.md](IMMEDIATE_SETPOINT_CHANGES.md)

---

## 📞 Support

- **Build issues**: Check [README.md](README.md)
- **Integration issues**: Check [REAL_GREENHOUSE_USAGE.md](REAL_GREENHOUSE_USAGE.md)
- **Theory questions**: Check [MATHEMATICS.md](MATHEMATICS.md)
- **Architecture questions**: Check [ARCHITECTURE_DIAGRAMS.md](ARCHITECTURE_DIAGRAMS.md)

---

**Last Updated**: October 9, 2026  
**Status**: ✅ Production Ready  
**Version**: 1.0

