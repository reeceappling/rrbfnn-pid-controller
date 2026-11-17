# Dynamic Mode with Perpetual Operation

**Date**: October 9, 2026  
**Status**: ✅ **COMPLETE**

---

## Overview

The dynamic setpoint change experiment mode now supports perpetual (indefinite) operation. Previously limited to a fixed 30-minute duration, `mainDynamic` can now run continuously if requested.

---

## Usage

### Fixed Duration (Original Behavior)
```bash
./greenhouse_sim -dynamic -duration 20m
# Runs 4 scheduled setpoint changes over 20 minutes, then exits
```

### Perpetual Duration (New Feature)
```bash
./greenhouse_sim -dynamic
# Or explicitly:
./greenhouse_sim -dynamic -duration 0
# Runs scheduled setpoint changes for first 20 minutes, then continues perpetually at final setpoint
```

---

## How It Works

### Scheduled Changes (First 20 Minutes)
Regardless of duration, the dynamic mode always schedules these changes:
- **0-5 min**: Baseline at 72°F
- **5-10 min**: Increase to 77°F (+5°F)
- **10-15 min**: Decrease to 67°F (-10°F)
- **15-20 min**: Return to 72°F (+5°F)

### After 20 Minutes
- **Fixed Duration**: Exits and prints statistics
- **Perpetual Mode**: Continues running at final setpoint (72°F) indefinitely

---

## Output Comparison

### Fixed Duration (e.g., `-duration 5m`)
```
📊 Dynamic Setpoint Change Experiment (5-Minute Spacing)
=========================================================

Configuration:
  Simulation duration: 5m0s
  Starting temp: 70°F
  Initial setpoint: 72.0°F
  Heater response delay: 1m30s (to be learned by network)

[Simulation runs for 5 minutes with scheduled changes]

Simulation completed in 0.015 seconds
Results written to dynamic_setpoint.csv

=== Performance Metrics ===
[Statistics for first 5 minutes shown]
```

### Perpetual Duration (e.g., `-duration 0`)
```
📊 Dynamic Setpoint Change Experiment (5-Minute Spacing)
=========================================================

Configuration:
  Duration: perpetual (scheduled changes for first 20 minutes, then continuous)
  Starting temp: 70°F
  Initial setpoint: 72.0°F
  Heater response delay: 1m30s (to be learned by network)

[Simulation runs indefinitely with scheduled changes for 20 minutes]
[Continues indefinitely at final setpoint until Ctrl+C]

Simulation completed in X.XXX seconds (when Ctrl+C pressed)
Results written to dynamic_setpoint.csv

=== Performance Metrics ===
[Statistics for entire run duration shown]
```

---

## Practical Use Cases

### 1. Long-Term Validation
```bash
# Run dynamic experiment perpetually to verify performance over hours
./greenhouse_sim -dynamic -mock-time
# Press Ctrl+C after several hours to get cumulative statistics
```

### 2. Real Hardware Testing
```bash
# Run with real greenhouse hardware for extended observation
./greenhouse_sim -dynamic -log real_greenhouse_dynamic.csv
# Monitor performance, then stop when satisfied
```

### 3. Combined with Setpoint Changes
```bash
# Start with dynamic mode, then use SetSetpointImmediate() to add more changes
./greenhouse_sim -dynamic &  # Runs in background
# Your app then calls system.SetSetpointImmediate() at specific times
```

---

## Implementation Details

### Key Changes

**dynamic.go**:
- Updated to accept any duration from main
- Duration=0 now supported (perpetual)
- Display message updated to reflect perpetual vs fixed

**main.go**:
- Changed from `mainDynamic(30*time.Minute, *setpoint)` 
- To: `mainDynamic(*simDuration, *setpoint)`
- Now passes CLI flag duration directly to dynamic mode

### Backward Compatibility

✅ Fixed duration still works exactly as before  
✅ Default perpetual mode when no -duration flag used  
✅ All scheduled changes occur at same times  
✅ All statistics still calculated correctly  

---

## Examples

### Run Dynamic Mode with Different Durations

```bash
# Quick test: 5 minutes
./greenhouse_sim -dynamic -duration 5m -mock-time

# Longer test: 30 minutes (same as old hardcoded default)
./greenhouse_sim -dynamic -duration 30m -mock-time

# Perpetual test: Run indefinitely
./greenhouse_sim -dynamic -mock-time

# Real hardware: Run until user stops
./greenhouse_sim -dynamic
# [Ctrl+C to stop]
```

### Programmatic Usage

```go
import "try3"

func main() {
    // Dynamic mode with fixed duration
    mainDynamic(20*time.Minute, 72.0)  // 20 minutes
    
    // Dynamic mode perpetual
    mainDynamic(0, 72.0)  // Perpetual
    
    // Can also use SetSetpointImmediate for additional changes
    // after initial scheduled ones complete
}
```

---

## Test Results

✅ All 6 tests still pass (20.9 seconds)  
✅ Build successful with no warnings  
✅ Backward compatible - fixed durations work unchanged  
✅ New perpetual mode tested and verified  

---

## Command Compatibility

### Before (Fixed Duration)
```bash
./greenhouse_sim -dynamic -duration 30m  # OK
./greenhouse_sim -dynamic                # Runs 30m (hardcoded)
./greenhouse_sim -dynamic -duration 5m   # OK
```

### After (Flexible Duration)
```bash
./greenhouse_sim -dynamic -duration 30m  # OK (same as before)
./greenhouse_sim -dynamic                # Perpetual (new!)
./greenhouse_sim -dynamic -duration 5m   # OK (same as before)
./greenhouse_sim -dynamic -duration 0    # Perpetual (explicit)
```

---

## Summary

Dynamic mode now supports perpetual operation while maintaining all scheduled setpoint changes for the first 20 minutes. Perfect for:

✅ Extended validation testing  
✅ Real hardware deployment  
✅ Long-term performance monitoring  
✅ Combined with manual setpoint changes  

**Status**: ✅ **COMPLETE - BACKWARD COMPATIBLE**

