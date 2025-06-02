// Package logger for sending log events to the application.
package logger

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/skpr/compass/cli/app/events"
)

// Logger for sending events into the CLI application.
type Logger struct {
	program *tea.Program
}

// New logger for sending events into the CLI application.
func New(program *tea.Program) (*Logger, error) {
	if program == nil {
		return nil, fmt.Errorf("program not provided")
	}

	return &Logger{
		program: program,
	}, nil
}

// Info message being logged to the application.
func (l *Logger) Info(_ string, _ ...any) {
	// We don't log info events.
}

// Debug message being logged to the application.
func (l *Logger) Debug(_ string, _ ...any) {
	// We don't log debug events.
}

// Error message being logged to the application.
func (l *Logger) Error(msg string, _ ...any) {
	l.send(msg, "error")
}

// Send log events to the CLI application.
func (l *Logger) send(msg, msgType string) {
	log := events.Log{
		Time:    time.Now(),
		Type:    msgType,
		Message: msg,
	}

	l.program.Send(log)
}
