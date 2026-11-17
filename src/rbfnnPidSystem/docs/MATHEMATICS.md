# Mathematical Foundations: Recurrent RBF Network for Adaptive PID Control

## Problem Formulation

### Greenhouse Dynamics Model

The greenhouse temperature evolves according to:

```
dT/dt = (Q_heater + Q_ambient - Q_loss) / C_th

Where:
  T(t)           = Interior temperature (°F)
  Q_heater       = Heat from heater [power 0-1]
  Q_ambient      = Heat from ambient conditions
  Q_loss         = Heat loss to environment = k(T - T_ambient)
  C_th           = Thermal mass (heat capacity)
  τ_delay        = Heater response delay (30-180 seconds)
```

The heater has a time delay, so the actual heater effect is:

```
T_eff(t) = T_cmd(t - τ_delay)
```

This is modeled via a circular buffer that stores past heater commands.

### Humidity Dynamics (Simplified)

```
dRH/dt = α(RH_ambient - RH) + β·T_heater - γ·RH_loss
```

Where the heater reduces relative humidity through evaporative cooling.

### Control Objective

Find adaptive PID gains K_p(t), K_i(t), K_d(t) such that:

```
min E[T_error²(t)] subject to:
  - 0 ≤ u(t) ≤ 1  (heater power bounds)
  - Computational efficiency (learn online in real-time)
```

## Recurrent RBF Network Architecture

### Input Layer

The augmented input vector is 41-dimensional:

```
z(t) ∈ ℝ^49 = [x(t); φ(t-1)]

Where:
  x(t) ∈ ℝ^41  = External input features (described below)
  φ(t-1) ∈ ℝ^8 = Previous hidden layer activations (recurrent)
```

**External Inputs x(t)** [41 dimensions]:
- 5 time taps (current, 1min ago, 2min, 3min, 4min ago) × 7 features:
  - Heater power u(t) ∈ [0,1]
  - Fogger power (unused)
  - Fan power (unused)
  - Interior temperature T_in(t) normalized to [-1,1]
  - Ambient temperature T_amb(t) normalized to [-1,1]
  - Interior humidity RH_in(t) normalized to [-1,1]
  - Ambient humidity RH_amb(t) normalized to [-1,1]
- 6 additional features (PID state):
  - Scaled temperature error e(t)/5
  - Scaled derivative de/dt
  - Scaled integral ∫e dt · K_i
  - Humidity error
  - Temperature gradient (T_in - T_amb)
  - Reserve dimension

**Normalization**:
```
x_normalized = (x_raw - center) / scale

Examples:
  T_norm = (T - 67.5°F) / 17.5  → output in [-1, 1]
  RH_norm = (RH - 0.7) / 30     → output in [-1, 1]
```

### Hidden Layer: Radial Basis Functions

Each hidden node j implements a Gaussian basis function:

```
φ_j(z) = exp(-||z - c_j||² / (2σ_j²))

Where:
  c_j ∈ ℝ^49     = Center vector of node j
  σ_j > 0        = Spread (width) parameter
  ||·||          = Euclidean norm
```

### Output Layer: Sigmoid-Bounded Gains

Raw output:
```
s_i = b_i + Σ_j w_ij · φ_j(z)    for i ∈ {1,2,3} (Kp, Ki, Kd)
```

Apply sigmoid squashing and scale to gain bounds:
```
K_i = B_i^min + (B_i^max - B_i^min) · sigmoid(s_i)

Where sigmoid(x) = 1 / (1 + exp(-x))

Gain bounds:
  B_Kp = [0.0, 1.0]
  B_Ki = [0.0, 0.5]
  B_Kd = [0.0, 0.3]
```

## Online Learning via Backpropagation

### Forward Pass

1. Augment input: `z = [x; φ_{prev}]`
2. Compute hidden activations: `φ_j(z) = exp(-dist²/(2σ²))`
3. Compute outputs: `s_i = b_i + Σ_j w_ij · φ_j`
4. Apply sigmoid and scale: `K_i = bound_scale(sigmoid(s_i))`

### Backward Pass

**Error signal** from greenhouse model:
```
e_control = T_setpoint - T_actual
```

**Chain rule through output layer**:
```
For each output i (Kp, Ki, Kd):
  
  δ_i = e_control · dE/du · du/dK_i · dK_i/ds_i
  
  Where:
    dE/du ≈ -1.0              (plant Jacobian: heater reduces +error)
    du/dK_i = u_i_component   (PID output decomposition)
    dK_i/ds_i = σ'(s_i) · (B_max - B_min)
    σ'(x) = σ(x)(1-σ(x))      (sigmoid derivative)
```

**Weight gradient** (output layer):
```
dJ/dw_ij = -δ_i · φ_j
w_ij ← w_ij - η_w · δ_i · φ_j
```

**Bias gradient**:
```
dJ/db_i = -δ_i
b_i ← b_i - η_w · δ_i
```

**Center gradient** (RBF layer):
```
For each node j:
  backErr = Σ_i δ_i · w_ij    (backpropagated error)
  
  For each dimension k:
    dJ/dc_jk = backErr · φ_j · (z_k - c_jk) / σ_j²
    c_jk ← c_jk - η_c · backErr · φ_j · (z_k - c_jk) / σ_j²
```

**Spread gradient**:
```
dist² = ||z - c_j||²

dJ/dσ_j = backErr · φ_j · dist² / σ_j³
σ_j ← σ_j - η_σ · backErr · φ_j · dist² / σ_j³
σ_j ← max(σ_j, 0.05)    [minimum spread floor]
```

### Learning Rates

Empirically determined (tunable):
```
η_w (weights):     0.01
η_c (centers):     0.005
η_σ (spreads):     0.002
```

## Structural Adaptation: Dynamic Growth

### Growth Criterion

A new RBF node is added when BOTH conditions hold:

```
1. |e_control| > ε_error  [High error in current region]
2. min_j ||z - c_j|| > ε_dist  [Input is novel/far from existing centers]

Thresholds:
  ε_error = 1.0°F
  ε_dist = 0.5 (normalized distance units)
```

### Growing a Node

When a new node is added, we must expand the center vector dimensionality:

```
Current state:
  - M existing nodes with centers in ℝ^49
  - prevActivations ∈ ℝ^8

Growing:
  1. Expand all existing centers to ℝ^50:
     For each node j: c_j ← [c_j; 0.5]  (0.5 = neutral)
  
  2. Create new center at current augmented input:
     c_new ← [z; 1.0]  (1.0 because it fires ~100% on itself)
  
  3. Expand recurrent state:
     φ_prev ← [φ_prev; φ_new]  (length 8 → 9)
  
  4. Add output weights:
     w_i,M+1 ← 0.0  (start neutral, gradient will adjust)
  
  5. Set spread based on nearest neighbor:
     σ_new = max(min_dist × 0.7, 0.3 × base_sigma)
     base_sigma = √(2D / M)  where D = 50, M = M+1
```

### Result

Network grows incrementally as it encounters new operating regimes:
- Early stages: 8 hidden nodes
- After 3-5 minutes: 12-17 nodes typical
- Maximum: 20 nodes (configurable limit)

## Integral State Management

### Anti-Windup Integration

The integral error accumulates with exponential forgetting:

```
e_int(t+dt) = e_int(t) · exp(-dt/τ_mem) + e_control(t) · dt

Where τ_mem = 5 minutes (memory time constant)

This prevents integral windup when setpoint or disturbance changes.
```

When PID output saturates:
```
if u > u_max:
    u = u_max
    e_int ← e_int - e·dt    [Reduce integral]
if u < u_min:
    u = u_min
    e_int ← e_int - e·dt
```

### Low-Pass Filtered Derivative

Raw derivative: `de/dt_raw = (e_k - e_{k-1}) / dt`

Exponential filter to reduce noise:
```
de/dt_filtered = α · de/dt_raw + (1-α) · de/dt_filtered_prev
α = 0.3  (30% raw, 70% history)
```

## Time-Delay Modeling via Circular Buffer

### Delay Buffer

A fixed-size circular buffer stores past heater commands:

```
Buffer size = ceil(τ_delay × sampling_rate)
            = ceil(90s × 2 Hz) = 180 slots

At each step:
  1. Store current heater command: buf[idx] = u(t)
  2. Retrieve delayed command: u_eff = buf[idx]  (from N steps ago)
  3. Advance index: idx = (idx + 1) % size
  4. Apply delayed command to greenhouse: dT/dt = f(u_eff)
```

This realizes the transport delay without requiring differential equations.

## Loss Function

Total accumulated squared error:

```
L(t) = L(t-dt) + e_control²(t)

Where e_control(t) = T_setpoint - T_actual(t)
```

Normalized MSE reported in statistics:
```
RMSE = √(L_total / n_steps)
```

## Convergence Properties

**No theoretical guarantees**, but empirically:

1. **Network size** stabilizes after ~5 minutes
2. **Gains** converge within ±5-10% after 3-5 minutes
3. **Steady-state error** typically <1-2°F with well-tuned parameters
4. **Overshoot** minimal due to adaptive Kd term

## Limitations & Considerations

### Recurrent Dimension Explosion

As network grows, the input dimension increases: 41 + M (num_hidden)

After adding 10 nodes: z ∈ ℝ^51, center computation overhead grows O(D²).

**Mitigation**: Upper limit on nodes (20 default) + pruning (disabled by default).

### Sparse Training Signal

In early control steps, error signal is weak (small initial tracking error).
Learning accelerates as error increases.

### Sensor Noise Impact

Gaussian noise on sensors: ±0.3°F, ±1.5% RH.
Derivative term amplifies noise → low-pass filter essential.

## References

### RBF Networks
- Poggio & Girosi (1990): "Networks for approximation and learning"
- Moody & Darken (1989): "Fast learning in networks of locally-tuned processing units"

### Adaptive Control
- Åström & Hägglund (2006): "Advanced PID Control"
- Narendra & Annaswamy (2012): "Stable Adaptive Systems"

### Recurrent Neural Networks
- Williams & Zipser (1989): "A learning algorithm for continually running fully recurrent neural networks"
- Hopfield (1982): "Neural networks and physical systems with emergent collective computational abilities"

