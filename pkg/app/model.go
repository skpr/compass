// Package app for handling the main application.
package app

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/table"

	"github.com/skpr/compass/pkg/app/events"
)

// DefaultMaxTraces is how many traces we retain when no limit has been configured.
const DefaultMaxTraces = 500

// NewModel for executing this application.
func NewModel(probePath string, maxTraces int) *Model {
	if maxTraces <= 0 {
		maxTraces = DefaultMaxTraces
	}

	return &Model{
		ProbePath: probePath,
		MaxTraces: maxTraces,
	}
}

// Model for storing the state of the application.
type Model struct {
	// Path to the compass.so we are probing.
	ProbePath string

	// MaxTraces is the maximum number of traces we retain, oldest are evicted first.
	MaxTraces int

	// The current display that is selected.
	PageSelected Page

	// Dimensions.
	Height int
	Width  int

	// Storage. Retained traces live in the search list, which is capped at
	// MaxTraces, so there is no second copy of them to keep in sync.
	Current *events.Trace

	// State of the connection to the trace stream.
	connection events.Connection

	// Models.
	search   list.Model
	metadata table.Model
	spans    table.Model
	totals   table.Model
	logs     list.Model
}
