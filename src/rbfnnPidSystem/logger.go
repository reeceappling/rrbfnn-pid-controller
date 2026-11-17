package coreSystem

import (
	"fmt"
	"os"
)

// Logger writes simulation data to CSV
type Logger struct {
	file   *os.File
	buffer []LogEntry
	maxBuf int
}

// NewLogger creates a new CSV logger
func NewLogger(filename string) *Logger {
	f, err := os.Create(filename)
	if err != nil {
		panic(fmt.Sprintf("Failed to create log file: %v", err))
	}

	logger := &Logger{
		file:   f,
		buffer: make([]LogEntry, 0, 1000),
		maxBuf: 1000,
	}

	// Write CSV header
	logger.writeHeader()
	return logger
}

// writeHeader writes the CSV column headers
func (l *Logger) writeHeader() {
	header := "Time,Setpoint,ActualTemp,Error,HeaterPower,Kp,Ki,Kd,ExternalTemp,ExternalHumidity,Humidity,Loss,NodeCount\n"
	l.file.WriteString(header)
}

// Log records a data point (buffered)
func (l *Logger) Log(entry LogEntry) {
	l.buffer = append(l.buffer, entry)
	if len(l.buffer) >= l.maxBuf {
		l.Flush()
	}
}

// Flush writes buffered data to disk
func (l *Logger) Flush() {
	for _, entry := range l.buffer {
		line := fmt.Sprintf("%s,%.3f,%.3f,%.3f,%.4f,%.6f,%.6f,%.6f,%.3f,%.3f,%.3f,%.6f,%d\n",
			entry.Timestamp.Format("15:04:05.000"),
			entry.Setpoint,
			entry.ActualTemp,
			entry.Error,
			entry.HeaterPower,
			entry.Kp,
			entry.Ki,
			entry.Kd,
			entry.ExternalTemp,
			entry.ExternalHumidity,
			entry.Humidity,
			entry.Loss,
			entry.NetworkNodeCount,
		)
		l.file.WriteString(line)
	}
	l.buffer = l.buffer[:0]
	l.file.Sync()
}

// Close flushes and closes the log file
func (l *Logger) Close() error {
	if len(l.buffer) > 0 {
		l.Flush()
	}
	return l.file.Close()
}
