#!/bin/bash
# Example usage script for the Recurrent RBF Greenhouse Control System

echo "=== Recurrent RBF Greenhouse Control Examples ==="
echo

# Build first if needed
echo "Building..."
go build -o greenhouse_sim  
if [ $? -ne 0 ]; then
    echo "Build failed!"
    exit 1
fi
echo "Build successful."
echo
chmod 777 greenhouse_sim

## Example 1: 5 minute mocked time
./greenhouse_sim -mock-time -duration 5m -log example_run_1.csv

## Example 2: 20 minute mocked time
./greenhouse_sim -mock-time -duration 20m -log example_run_2.csv

## Example 3: 1hr mocked time
./greenhouse_sim -mock-time -duration 60m -log example_run_3.csv

## Example 4: 5 minute mocked time dynamic setpoint
#./greenhouse_sim -dynamic -mock-time example_run_4.csv           # Dynamic setpoint test

## Example 5: 5 minute mocked time dynamic setpoint
#./greenhouse_sim -dynamic -mock-time -duration 5m example_run_5.csv        # 5-minute dynamic setpoint test (instant)





## Example 1: Quick test
#echo "Example 1: Quick 30-second test"
#echo "Command: ./try3 -duration=30s -setpoint=72 -delay=60s"
#echo "Description: Brief test to verify system works"
#echo
#./try3 -duration=30s -setpoint=72 -delay=60s -log=example1_30sec.csv
#echo

## Example 2: Standard 5-minute run
#echo "Example 2: Standard 5-minute control experiment"
#echo "Command: ./try3 -duration=5m -setpoint=72 -delay=90s -verbose=true"
#echo "Description: Typical run with 90s heater delay"
#echo
#./try3 -duration=5m -setpoint=72 -delay=90s -verbose=true -log=example2_5min.csv
#echo

## Example 3: Longer delay
#echo "Example 3: Longer delay (3 minute response lag)"
#echo "Command: ./try3 -duration=10m -setpoint=75 -delay=180s"
#echo "Description: Tests network's ability to handle longer time delays"
#echo
#./try3 -duration=10m -setpoint=75 -delay=180s -log=example3_longdelay.csv
#echo
#
## Example 4: Higher setpoint
#echo "Example 4: Higher setpoint temperature"
#echo "Command: ./try3 -duration=5m -setpoint=80 -delay=90s"
#echo "Description: Network learns to control at higher temperature"
#echo
#./try3 -duration=5m -setpoint=80 -delay=90s -log=example4_hightemp.csv
#echo
#
#echo "=== Examples Complete ==="
#echo "Check the generated .csv files for results:"
#ls -lah example*.csv

