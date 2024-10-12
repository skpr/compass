// Package app for handling the main application.
package app

import (
	"github.com/skpr/compass/collector/pkg/tracing/complete"
)

// Model for storing the state of the application.
type Model struct {
	// This is how we track the height and width of the terminal.
	width int
	// This is how we track the height and width of the terminal.
	height int
	// Internal storage for profiles.
	profiles []complete.Profile
	// Index of the currently selected profile.
	profileSelected int
	// Index of the currently selected profile's scroll position.
	breakdownScroll int
}

// NewModel creates a new model.
func NewModel() Model {
	return Model{}
}
