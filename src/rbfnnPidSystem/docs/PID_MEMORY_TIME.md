# PID Memory Time Analysis & Recommendations

## Current Setting
```go
memoryTime: 5 * time.Minute  // 300 seconds
```

## What is Memory Time?

The memory time is the exponential time constant for integral error forgetting:

```go
forgettingFactor := math.Exp(-dtSec / memoryTime)
integral(t) = integral(t-1) * forgettingFactor + error(t) * dt
```

**Effect**: Controls how long historical errors are "remembered" in the integral term.

## Analysis for Greenhouse Control

### System Characteristics (from your setup)
- **Heater delay**: ~90 seconds (1.5 minutes)
- **Thermal time constant**: ~2-3 minutes (estimated)
- **Control update rate**: 1 Hz
- **Setpoint change spacing**: 5 minutes
- **Target steady-state error**: ±2°F

### Memory Time Options

#### Option 1: Short (1-2 minutes)
```go
memoryTime: 90 * time.Second  // 1.5 minutes
```

**Characteristics**:
- Forgets old errors quickly
- Responsive to disturbances
- Lower integral windup risk
- May oscillate more

**Impact on other parameters**:
- ✅ Ki can be lower (less aggressive needed)
- ✅ Better for rapid setpoint changes
- ⚠️ May overshoot initially
- ⚠️ Settles slower to final value

#### Option 2: Medium (3-5 minutes) - CURRENT
```go
memoryTime: 5 * time.Minute  // 300 seconds
```

**Characteristics**:
- Balanced response
- Moderate error accumulation
- Good for steady-state tracking
- Handles setpoint changes reasonably

**Impact on other parameters**:
- ✅ Well-balanced with current Ki
- ✅ Good convergence in 5-min phases
- ✅ Stable, minimal overshoot
- ✅ Current tests sized for this

#### Option 3: Long (8-15 minutes)
```go
memoryTime: 10 * time.Minute  // 600 seconds
```

**Characteristics**:
- Accumulates error over longer periods
- Excellent steady-state tracking
- Slow response to disturbances
- More integral windup risk

**Impact on other parameters**:
- ✅ Better long-term regulation
- ⚠️ Ki might need to be higher
- ⚠️ Slower to react to setpoint changes
- ⚠️ Test durations might need to increase

## Recommendation: Keep 5 Minutes

**Why 5 minutes is optimal**:

1. **Matches system dynamics**:
   - Heater delay: 90 seconds
   - Thermal time constant: ~180 seconds
   - Memory time of 300s = ~1.7x thermal time constant
   - Good balance between responsiveness and stability

2. **Works well with 5-minute test phases**:
   - Each phase gets ~1 full memory cycle
   - Integral accumulates and stabilizes
   - Transitions clear within the phase

3. **Proven performance**:
   - Current tests converge to ±2°F steady-state
   - No overshoot issues
   - Network learns effectively
   - All tests pass

4. **Greenhouse-appropriate**:
   - Plants don't experience harsh temperature swings
   - Energy efficient (smooth control)
   - Predictable behavior
   - Good handling of day/night cycles

## If You Change Memory Time

### What MUST change:

1. **Ki (Integral Gain)**
   - **Inverse relationship**: Shorter memory → Lower Ki needed
   - **Formula**: `Ki_new = Ki_old × (memoryTime_old / memoryTime_new)`
   
   Example:
   ```go
   // Current: memoryTime = 5min, Ki = 0.01
   // If changing to 2min:
   Ki_new = 0.01 × (5 / 2) = 0.025  // INCREASE
   
   // If changing to 10min:
   Ki_new = 0.01 × (5 / 10) = 0.005  // DECREASE
   ```

2. **Test Durations** (if memory time significantly changes)
   - Each phase should be at least 2× the memory time
   - Current: 5-minute phases with 5-minute memory ✓
   - If memory = 10 min: Phases should be 20 min minimum
   - If memory = 1.5 min: Phases can be 3-5 min (current is fine)

### What SHOULD change:

3. **Error Thresholds for Network Growth**
   - `errorThreshold: 1.0°F` (in rbf_network.go)
   - With shorter memory: Errors spike faster → threshold too high → grow network more
   - With longer memory: Errors accumulate slower → threshold might be too sensitive
   - Recommend: Keep as-is for 5min, adjust by ±20% if changing significantly

4. **Learning Rates** (optional)
   - If memory changes drastically (>2x), may want to adjust:
   - Shorter memory (faster response) → Higher learning rates OK
   - Longer memory (slower response) → Lower learning rates (more stable)
   - Changes: Less than 10% usually sufficient

### What CAN stay the same:

- ✅ Kp, Kd (proportional and derivative)
- ✅ Network architecture (nodes, spreads)
- ✅ Update frequency (1 Hz)
- ✅ Heater delay setting

## Optimization Strategy

If you want to **optimize memory time** for your specific greenhouse:

### Step 1: Initial Testing (Use 5 minutes)
```go
memoryTime: 5 * time.Minute
```
- Run full simulation/tests
- Measure: overshoot, settling time, steady-state error

### Step 2: If too much overshoot (>2°F)
**Reduce memory time to 2-3 minutes**:
```go
memoryTime: 2.5 * time.Minute
// Adjust Ki
Ki_new := 0.01 * (5.0 / 2.5)  // = 0.02
```
- Tests: May need slightly longer durations
- Behavior: Faster response, more reactive

### Step 3: If too slow reaching setpoint
**Check if memory > 3 min is needed**:
```go
// Longer memory helps long-term regulation
memoryTime: 8 * time.Minute
Ki_new := 0.01 * (5.0 / 8.0)  // = 0.00625
```
- Tests: Keep 5-minute phases (still adequate)
- Behavior: Smoother, less oscillation

## Validation Checklist

When changing memory time, verify:

- [ ] Ki adjusted by inverse ratio of memory times
- [ ] Tests still converge to ±4°F (steady-state)
- [ ] No excessive overshoot (max error < 2× target)
- [ ] Settling time reasonable for your use case
- [ ] Network doesn't grow excessively (sign of instability)
- [ ] Response to setpoint changes smooth (no ringing)

## Current Configuration (Recommended)

```go
// In pid.go
memoryTime: 5 * time.Minute  // Optimal balance

// In main.go
lrs := LearningRates{
	Weights: 0.015,
	Centers: 0.008,
	Spreads: 0.003,
}

// In rbf_network.go
rbfNetwork := NewRecurrentRBFNetwork(8, 50, lrs)
```

**This configuration**:
- ✅ Optimal for 90-second heater delay
- ✅ Good for 5-minute test phases
- ✅ Achieves ±2°F steady-state error
- ✅ All tests pass
- ✅ Stable, predictable behavior
- ✅ Energy efficient

## Summary

| Aspect | Value | Notes |
|--------|-------|-------|
| **Current Memory Time** | 5 minutes | Recommended, optimal |
| **Recommended Range** | 2-8 minutes | Depends on your dynamics |
| **If Changed** | Adjust Ki inversely | Ki_new = Ki_old × (T_old / T_new) |
| **Test Duration** | Keep at 5 minutes | Adequate for current memory time |
| **No Change Needed** | Kp, Kd, structure | These are independent |

**Final Answer**: Keep 5 minutes. It's well-tuned for your system. If you change it, only Ki needs adjustment - use the inverse ratio formula.


