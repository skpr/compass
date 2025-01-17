// Package app for handling the main application.
package app

import (
	"github.com/skpr/compass/trace"
)

// ViewMode for analysing tracing data.
type ViewMode string

const (
	// ViewSpans will display the trace as spans.
	ViewSpans ViewMode = "spans"
	// ViewCount will provide a counted view of the function calls.
	ViewCount ViewMode = "count"
)

// Model for storing the state of the application.
type Model struct {
	// This is how we track the height and width of the terminal.
	width int
	// This is how we track the height and width of the terminal.
	height int
	// Internal storage for traces.
	traces Traces
	// View mode used to analyse traces.
	viewMode ViewMode
	// Index of the currently selected trace.
	traceSelected int
	// Index of the currently selected trace's scroll position.
	breakdownScroll int
	searchQuery     string
}

// Traces stored in this model.
type Traces struct {
	All      []trace.Trace
	Filtered []trace.Trace
}

// NewModel creates a new model.
func NewModel() *Model {
	return &Model{}
}
