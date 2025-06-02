// Package app for handling the main application.
package app

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/table"
	"github.com/skpr/compass/cli/app/events"
)

// NewModel for executing this application.
func NewModel(probePath string) *Model {
	return &Model{
		ProbePath: probePath,
	}
}

// Model for storing the state of the application.
type Model struct {
	// Path to the compass.so we are probing.
	ProbePath string

	// The current display that is selected.
	PageSelected Page

	// Dimensions.
	Height int
	Width  int

	// Storage.
	Current *events.Trace
	Traces  map[string]events.Trace

	// Models.
	search   list.Model
	metadata table.Model
	spans    table.Model
	totals   table.Model
	logs     list.Model
}
