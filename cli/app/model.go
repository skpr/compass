// Package app for handling the main application.
package app

import (
	"github.com/skpr/compass/trace"
)

// Model for storing the state of the application.
type Model struct {
	// This is how we track the height and width of the terminal.
	width int
	// This is how we track the height and width of the terminal.
	height int
	// Internal storage for traces.
	traces []trace.Trace
	// Index of the currently selected trace.
	traceSelected int
	// Index of the currently selected trace's scroll position.
	breakdownScroll int
}

// NewModel creates a new model.
func NewModel() Model {
	return Model{}
}
