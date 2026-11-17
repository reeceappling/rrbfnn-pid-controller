# Heater Delay Learning - Implementation Details

## Design Philosophy

The heater delay is **NOT passed as a parameter to the network**. Instead:

1. **The simulator has a fixed delay** (configurable via `-delay` flag)
2. **The network learns this delay intrinsically** through:
   - Recurrent hidden layer state that captures temporal dynamics
   - Online learning from error signals
   - Circular buffer in simulator that models the actual delay

## How the Network Learns the Delay

### Mechanism
```
t=0: User commands heater power = 1.0
t=90s: Greenhouse temperature starts to increase (delay effect)
       Network receives: "I commanded heater, but temp hasn't changed yet"
       
t=180s: Greenhouse continues heating
        Network receives: "Now temp is changing, linking it to command 90s ago"
        
Through recurrent connections, the network learns:
  - Previous activations encode "what happened before"
  - Current error shows delayed effects
  - Backpropagation teaches centers/spreads/weights
  - Over many cycles, temporal patterns emerge
```

### Key Components

1. **Recurrent Activations**: Previous hidden layer outputs fed back as inputs
   - Effectively creates a memory of past states
   - Dimensionality grows: 41 (external) + 8-20 (recurrent) = 49-61 total

2. **Error Signals**: Temperature error = setpoint - measured_temp
   - Shows delayed response to heater commands
   - Network backpropagates this to learn delay patterns

3. **Online Learning**: Every step updates weights/centers/spreads
   - No batch training required
   - Learns continuously as system operates
   - Adapts to different delay values

4. **Circular Buffer**: Fixed-size delay in simulator
   - Applies heater command from N timesteps ago
   - Network doesn't know the size N, only sees the effects

## Usage

### Default (90-second delay)
```bash
./try3 -dynamic
# Network learns 90-second heater delay
```

### Custom delay (2-minute)
```bash
./try3 -dynamic -delay=120s
# Network learns 120-second heater delay without being told
```

### Different duration and setpoint
```bash
./try3 -dynamic -duration=20m -setpoint=75 -delay=60s
# Network learns to control at 75°F with 60s delay
```

## Why This Design?

### Advantages
1. **More realistic**: Real systems rarely know their exact delay value
2. **Robust**: Network learns to handle whatever delay is present
3. **Generalizable**: Network trained on one delay can adapt to new delays through recurrent learning
4. **Educational**: Demonstrates how online learning captures temporal dynamics

### How It Differs From Hand-Tuning
- **Traditional PID**: Requires explicit delay specification (e.g., Ziegler-Nichols)
- **Our RBF Network**: Discovers delay through error-driven learning
- **Advantage**: No manual tuning needed, automatic adaptation

## Evidence of Learning

The network demonstrates it learned the delay through:

1. **Network Growth**: Adds nodes as it encounters new temporal patterns
2. **Gain Evolution**: Kp, Ki, Kd change over time to compensate for learned delay
3. **Error Reduction**: Mean error decreases as network models dynamics better
4. **Stable Control**: Despite not knowing the delay, system stabilizes

## Comparison: With vs Without Recurrent Connections

### Without Recurrence (Feedforward)
```
Input: [external_features_only]
Problem: Can't see delay, treats each timestep independently
Result: Poor adaptation to delayed systems
```

### With Recurrence (Our Implementation)
```
Input: [external_features + previous_hidden_activations]
Advantage: Temporal memory encoded in recurrent state
Result: Network learns delay dynamics implicitly
```

## Mathematical Perspective

The recurrent RBF effectively learns a **nonlinear state-space model**:

```
φ(t+1) = basis_functions([x(t); φ(t)])    // Hidden activations
y(t) = linear_combination(φ(t+1))          // Output (PID gains)

Where:
  φ(t) = previous hidden activations (temporal memory)
  x(t) = current external inputs
  The circular delay in simulator creates the temporal relationship
```

Over time, backpropagation adjusts centers and spreads so that the basis functions capture:
- Short-term dynamics (immediate heater response)
- Medium-term dynamics (temperature rise with delay)
- Long-term dynamics (steady-state thermal balance)

## Testing the Learning

Run the dynamic experiment:
```bash
./try3 -dynamic
```

Observe:
1. **Phase 0** (0-5min at 72°F): Network learns baseline dynamics
2. **Phase 1** (5-10min at 77°F): Network adapts gains for higher demand
3. **Phase 2** (10-15min at 67°F): Network learns cooling response (different delay characteristics)
4. **Phase 3** (15-20min at 72°F): Network returns to familiar operating point

The fact that it controls well across all phases proves it learned the temporal dynamics.

## Future Possibilities

- **Adaptive delay learning**: If actual delay changes mid-operation
- **Multi-delay systems**: Multiple actuators with different response times
- **Disturbance compensation**: Learning external temp effects
- **Predictive control**: Using learned model for look-ahead decisions

## Conclusion

The network **does not need to know the heater delay explicitly**. Through its recurrent connections and online learning from error signals, it discovers and adapts to the temporal dynamics of the greenhouse system automatically.

**Status**: ✅ Heater delay is learned intrinsically, not hardcoded or explicitly passed

