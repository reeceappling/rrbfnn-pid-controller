# How to Use NewRealGreenhouseEnvironment

**Date**: October 9, 2026  
**Purpose**: Guide for integrating real greenhouse hardware with the RBF control system

---

## Overview

`NewRealGreenhouseEnvironment` creates a bridge between the RBF control system and real physical greenhouse hardware. Instead of using the simulator, it reads from actual sensors and controls actual heater actuators.

**Function Signature:**
```go
func NewRealGreenhouseEnvironment(
    sensors HardwareSensors,
    actuators HardwareActuators,
    sensorUpdateInterval time.Duration,
    actuatorUpdateInterval time.Duration,
) (*RealGreenhouseEnvironment, error)
```

---

## Step 1: Implement the HardwareSensors Interface

You must implement this interface to read from your actual sensors:

```go
type HardwareSensors interface {
    // ReadTemperatureHumidity reads from your sensors
    // Returns: internalTemp(°F), internalHumidity(%), 
    //          externalTemp(°F), externalHumidity(%), error
    ReadTemperatureHumidity() (float64, float64, float64, float64, error)
    
    // Initialize sensor hardware
    Initialize() error
    
    // Cleanup sensor resources
    Close() error
}
```

### Example 1: DHT22 Sensors via GPIO

```go
import (
    "periph.io/x/conn/v3/physic"
    "periph.io/x/devices/v3/dht"
    "periph.io/x/host/v3"
)

type DHTSensors struct {
    internalSensor *dht.Dev  // GPIO pin 27
    externalSensor *dht.Dev  // GPIO pin 22
}

func NewDHTSensors() (*DHTSensors, error) {
    // Initialize GPIO
    if _, err := host.Init(); err != nil {
        return nil, err
    }
    
    // Open pins (GPIO 27 for internal, GPIO 22 for external)
    // This is pseudo-code; actual implementation depends on your wiring
    
    return &DHTSensors{
        internalSensor: internalSensor,
        externalSensor: externalSensor,
    }, nil
}

func (d *DHTSensors) ReadTemperatureHumidity() (float64, float64, float64, float64, error) {
    // Read internal sensor
    env := physic.Env{}
    if err := d.internalSensor.Sense(&env); err != nil {
        return 0, 0, 0, 0, err
    }
    internalTemp := float64(env.Temperature) / 1e9 * 1.8 + 32  // Convert to °F
    internalHumidity := float64(env.Humidity) / 1e6 * 100      // Convert to %
    
    // Read external sensor
    env = physic.Env{}
    if err := d.externalSensor.Sense(&env); err != nil {
        return 0, 0, 0, 0, err
    }
    externalTemp := float64(env.Temperature) / 1e9 * 1.8 + 32
    externalHumidity := float64(env.Humidity) / 1e6 * 100
    
    return internalTemp, internalHumidity, externalTemp, externalHumidity, nil
}

func (d *DHTSensors) Initialize() error {
    // Already initialized in NewDHTSensors
    return nil
}

func (d *DHTSensors) Close() error {
    // Clean up resources if needed
    return nil
}
```

### Example 2: I2C Sensors (BME680, BMP390)

```go
import (
    "periph.io/x/conn/v3/i2c/i2creg"
    "periph.io/x/devices/v3/bme680"
)

type BME680Sensors struct {
    internalSensor *bme680.Dev
    externalSensor *bme680.Dev
}

func NewBME680Sensors(internalI2CAddr, externalI2CAddr uint16) (*BME680Sensors, error) {
    // Open I2C bus
    bus, err := i2creg.Open("")
    if err != nil {
        return nil, err
    }
    
    // Connect to sensors at different addresses
    internalSensor, err := bme680.New(bus, &bme680.Opts{Addr: internalI2CAddr})
    if err != nil {
        return nil, err
    }
    
    externalSensor, err := bme680.New(bus, &bme680.Opts{Addr: externalI2CAddr})
    if err != nil {
        return nil, err
    }
    
    return &BME680Sensors{
        internalSensor: internalSensor,
        externalSensor: externalSensor,
    }, nil
}

func (b *BME680Sensors) ReadTemperatureHumidity() (float64, float64, float64, float64, error) {
    env := physic.Env{}
    if err := b.internalSensor.Sense(&env); err != nil {
        return 0, 0, 0, 0, err
    }
    
    internalTemp := float64(env.Temperature) / 1e9 * 1.8 + 32
    internalHumidity := float64(env.Humidity) / 1e6 * 100
    
    env = physic.Env{}
    if err := b.externalSensor.Sense(&env); err != nil {
        return 0, 0, 0, 0, err
    }
    
    externalTemp := float64(env.Temperature) / 1e9 * 1.8 + 32
    externalHumidity := float64(env.Humidity) / 1e6 * 100
    
    return internalTemp, internalHumidity, externalTemp, externalHumidity, nil
}

func (b *BME680Sensors) Initialize() error {
    return nil
}

func (b *BME680Sensors) Close() error {
    return nil
}
```

---

## Step 2: Implement the HardwareActuators Interface

You must implement this interface to control your heater:

```go
type HardwareActuators interface {
    // SetHeaterPower sets heater power (0.0 = off, 1.0 = full)
    SetHeaterPower(power float64) error
    
    // GetHeaterStatus returns current power level
    GetHeaterStatus() (float64, error)
    
    // Initialize actuator hardware
    Initialize() error
    
    // Cleanup actuator resources
    Close() error
    
    // EmergencyShutdown stops all actuators immediately
    EmergencyShutdown() error
}
```

### Example 1: PWM Heater Control

```go
import (
    "periph.io/x/conn/v3/gpio"
    "periph.io/x/conn/v3/gpio/gpioreg"
)

type PWMHeater struct {
    pwmPin     gpio.PinIO
    currentPower float64
}

func NewPWMHeater(gpioPin string) (*PWMHeater, error) {
    pin, err := gpioreg.ByName(gpioPin)  // e.g., "GPIO18"
    if err != nil {
        return nil, err
    }
    
    return &PWMHeater{
        pwmPin: pin,
        currentPower: 0.0,
    }, nil
}

func (h *PWMHeater) SetHeaterPower(power float64) error {
    // Clamp to [0, 1]
    if power < 0.0 {
        power = 0.0
    }
    if power > 1.0 {
        power = 1.0
    }
    
    // Set PWM duty cycle (0% to 100%)
    dutyCycle := uint32(power * 100)
    
    // This is pseudo-code; actual PWM control depends on library
    // h.pwmPin.PWM(frequency, dutyCycle)
    
    h.currentPower = power
    return nil
}

func (h *PWMHeater) GetHeaterStatus() (float64, error) {
    return h.currentPower, nil
}

func (h *PWMHeater) Initialize() error {
    // Setup PWM frequency (e.g., 1kHz)
    return nil
}

func (h *PWMHeater) Close() error {
    // Turn off heater
    return h.SetHeaterPower(0.0)
}

func (h *PWMHeater) EmergencyShutdown() error {
    return h.SetHeaterPower(0.0)
}
```

### Example 2: Relay-Based Heater Control

```go
type RelayHeater struct {
    relayPin      gpio.PinIO
    currentPower  float64
    pwmCycle      int  // For approximate PWM via relay
}

func NewRelayHeater(gpioPin string) (*RelayHeater, error) {
    pin, err := gpioreg.ByName(gpioPin)
    if err != nil {
        return nil, err
    }
    
    // Set as output
    err = pin.Out(gpio.Low)
    if err != nil {
        return nil, err
    }
    
    return &RelayHeater{
        relayPin: pin,
        currentPower: 0.0,
        pwmCycle: 0,
    }, nil
}

func (h *RelayHeater) SetHeaterPower(power float64) error {
    // Clamp
    if power < 0.0 {
        power = 0.0
    }
    if power > 1.0 {
        power = 1.0
    }
    
    // For relay, approximate PWM with duty cycle
    // Full power (> 0.5): relay on
    // Low power (≤ 0.5): relay pulses
    if power > 0.9 {
        // Full on
        h.relayPin.Out(gpio.High)
    } else if power > 0.1 {
        // PWM approximation: cycle the relay
        // This would typically be done via a background goroutine
        h.pwmCycle = int(power * 100)  // Store for background control
    } else {
        // Off
        h.relayPin.Out(gpio.Low)
    }
    
    h.currentPower = power
    return nil
}

func (h *RelayHeater) GetHeaterStatus() (float64, error) {
    return h.currentPower, nil
}

func (h *RelayHeater) Initialize() error {
    return nil
}

func (h *RelayHeater) Close() error {
    return h.SetHeaterPower(0.0)
}

func (h *RelayHeater) EmergencyShutdown() error {
    return h.SetHeaterPower(0.0)
}
```

---

## Step 3: Create the Environment

Once you have sensor and actuator implementations, create the environment:

```go
package main

import (
    "context"
    "log"
    "time"
)

func main() {
    // Create sensor implementation
    sensors, err := NewDHTSensors()
    if err != nil {
        log.Fatalf("Failed to initialize sensors: %v", err)
    }
    defer sensors.Close()
    
    // Create actuator implementation
    actuators, err := NewPWMHeater("GPIO18")
    if err != nil {
        log.Fatalf("Failed to initialize actuators: %v", err)
    }
    defer actuators.Close()
    
    // Create real greenhouse environment
    env, err := NewRealGreenhouseEnvironment(
        sensors,
        actuators,
        100*time.Millisecond,  // Read sensors every 100ms
        500*time.Millisecond,  // Update heater every 500ms
    )
    if err != nil {
        log.Fatalf("Failed to create environment: %v", err)
    }
    defer env.Close()
    
    // Verify hardware is working
    if err := env.HealthCheck(); err != nil {
        log.Fatalf("Hardware health check failed: %v", err)
    }
    
    log.Println("✓ Real greenhouse environment ready")
    log.Println(env.GetStatus())
}
```

---

## Step 4: Integrate with Control System

After creating the environment, you can use it with the control system. However, currently `RunGreenhouseSimulation` uses `GreenhouseSimulator`. You'll need to create a wrapper or modify the system to accept a generic interface.

### Option A: Create a Unified Interface

```go
// GreenhouseInterface abstracts both simulator and real environment
type GreenhouseInterface interface {
    ReadSensors() (GreenhouseInput, error)
    SendControl(output ControlOutput) error
    GetCurrentHeaterPower() float64
    Close() error
}

// Adapt RealGreenhouseEnvironment to this interface
func (r *RealGreenhouseEnvironment) ReadSensors() (GreenhouseInput, error) {
    // Implementation to convert sensor readings to GreenhouseInput
    reading, err := r.hardwareSensors.ReadTemperatureHumidity()
    if err != nil {
        return GreenhouseInput{}, err
    }
    
    return GreenhouseInput{
        InternalTemp:     reading.internalTemp,
        InternalHumidity: reading.internalHumidity,
        ExternalTemp:     reading.externalTemp,
        ExternalHumidity: reading.externalHumidity,
        Timestamp:        time.Now(),
    }, nil
}

func (r *RealGreenhouseEnvironment) SendControl(output ControlOutput) error {
    return r.hardwareActuators.SetHeaterPower(output.HeaterPower)
}
```

### Option B: Use with Parallel Control Loop

```go
func RunRealGreenhouseControl(
    ctx context.Context,
    env *RealGreenhouseEnvironment,
    config SimConfig,
) error {
    // Initialize control system components
    lrs := LearningRates{
        Weights: 0.015,
        Centers: 0.008,
        Spreads: 0.003,
    }
    
    rbfNetwork := NewRecurrentRBFNetwork(10, 50, lrs)
    rbfNetwork.SetGrowthEnabled(config.enableGrowth)
    rbfNetwork.SetPruningEnabled(config.enablePruning)
    
    pidController := NewAdaptivePIDController()
    
    logger := NewLogger(config.logFile)
    defer logger.Close()
    
    setpoint := config.setpoint
    ticker := time.NewTicker(1 * time.Second)
    defer ticker.Stop()
    
    startTime := time.Now()
    
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
            
        case <-ticker.C:
            // Read real sensors
            sensorData, err := env.ReadSensors()
            if err != nil {
                log.Printf("Sensor read error: %v", err)
                continue
            }
            
            // Compute control action
            error := sensorData.InternalTemp - setpoint
            
            // RBF network forward pass
            inputs := constructInputs(sensorData, error)
            rbfOut := rbfNetwork.Forward(inputs)
            
            // PID control
            pidOut := pidController.Compute(
                error,
                rbfOut.Kp,
                rbfOut.Ki,
                rbfOut.Kd,
                1*time.Second,
            )
            
            // Send to real actuators
            if err := env.SendControl(ControlOutput{
                HeaterPower: pidOut.HeaterPower,
            }); err != nil {
                log.Printf("Actuator error: %v", err)
                // Continue; failsafe should handle this
            }
            
            // Learn online
            rbfNetwork.BackwardAndUpdate(error, rbfOut.HiddenActivations, rbfOut.AugmentedInputs, rbfOut.PreSigmoidRaw)
            rbfNetwork.AdaptStructure(inputs, error)
            
            // Log results
            logger.Log(LogEntry{
                Timestamp:        sensorData.Timestamp,
                Setpoint:         setpoint,
                ActualTemp:       sensorData.InternalTemp,
                Error:            error,
                HeaterPower:      pidOut.HeaterPower,
                Kp:               rbfOut.Kp,
                Ki:               rbfOut.Ki,
                Kd:               rbfOut.Kd,
                ExternalTemp:     sensorData.ExternalTemp,
                ExternalHumidity: sensorData.ExternalHumidity,
                Humidity:         sensorData.InternalHumidity,
            })
        }
    }
}
```

---

## Best Practices

### 1. Always Check Hardware Health

```go
if err := env.HealthCheck(); err != nil {
    log.Fatalf("Hardware not responding: %v", err)
}
```

### 2. Implement Proper Error Handling

```go
if err := env.SendControl(output); err != nil {
    log.Printf("Control error: %v, entering failsafe mode", err)
    // Set conservative defaults
    pidController.Reset()
}
```

### 3. Use Emergency Stop When Needed

```go
// On critical error
if criticalError {
    env.EmergencyStop()
    log.Fatal("EMERGENCY STOP")
}
```

### 4. Monitor Hardware Status

```go
// Periodically check health
go func() {
    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()
    for range ticker.C {
        if err := env.HealthCheck(); err != nil {
            log.Printf("Hardware health check failed: %v", err)
        }
    }
}()
```

### 5. Log Status Regularly

```go
status := env.GetStatus()
log.Printf(
    "Temp: %.1f°F, RH: %.1f%%, Heater: %.1f%%, Reads: %d, Writes: %d",
    status["internal_temp"],
    status["internal_humidity"],
    status["heater_power"],
    status["total_sensor_reads"],
    status["total_actuator_writes"],
)
```

---

## Troubleshooting

### Sensors Not Responding
```go
if err := env.HealthCheck(); err != nil {
    log.Println("Check I2C/GPIO connections")
    log.Println("Run `gpio readall` or `i2cdetect -y 1` to diagnose")
}
```

### Heater Not Turning On
- Check PWM pin configuration
- Verify relay wiring
- Test with `SetHeaterPower(1.0)` directly
- Use multimeter to verify voltage

### Sensor Readings Inconsistent
- Check sensor calibration
- Verify power supply stability
- Add filtering/averaging

---

## Complete Example: Raspberry Pi Greenhouse

```go
package main

import (
    "context"
    "log"
    "time"
)

func main() {
    // Setup sensors (DHT22 on GPIO 27 and GPIO 22)
    sensors, err := NewDHTSensors()
    if err != nil {
        log.Fatalf("Sensor init: %v", err)
    }
    defer sensors.Close()
    
    // Setup heater (PWM on GPIO 18)
    actuators, err := NewPWMHeater("GPIO18")
    if err != nil {
        log.Fatalf("Actuator init: %v", err)
    }
    defer actuators.Close()
    
    // Create environment
    env, err := NewRealGreenhouseEnvironment(
        sensors,
        actuators,
        500*time.Millisecond,
        500*time.Millisecond,
    )
    if err != nil {
        log.Fatalf("Environment init: %v", err)
    }
    defer env.Close()
    
    // Verify hardware
    if err := env.HealthCheck(); err != nil {
        log.Fatalf("Health check: %v", err)
    }
    
    log.Println("✓ Real greenhouse system ready")
    
    // Run control loop
    ctx, cancel := context.WithTimeout(
        context.Background(),
        24*time.Hour,  // Run for 24 hours
    )
    defer cancel()
    
    err = RunRealGreenhouseControl(ctx, env, SimConfig{
        simDuration:   24*time.Hour,
        setpoint:      72.0,
        heaterDelay:   90*time.Second,
        logFile:       "greenhouse_real.csv",
        verbose:       true,
        useMockTime:   false,
        enableGrowth:  true,
        enablePruning: false,
        seed:          42,
    })
    
    if err != nil {
        log.Fatalf("Control loop error: %v", err)
    }
}
```

---

## Summary

1. **Implement HardwareSensors** - Read temp/humidity
2. **Implement HardwareActuators** - Control heater
3. **Create RealGreenhouseEnvironment** - Bridge to hardware
4. **Integrate with control system** - Use with RBF + PID
5. **Handle errors gracefully** - Use health checks and failsafes
6. **Monitor performance** - Log status regularly

The system is now ready for real-world greenhouse deployment!

