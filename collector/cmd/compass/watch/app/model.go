package app

import "github.com/skpr/compass/collector/cmd/compass/watch/app/profile"

// Model for storing the state of the application.
type Model struct {
	// This is how we track the height and width of the terminal.
	width int
	// This is how we track the height and width of the terminal.
	height int
	// Internal storage for profiles.
	profiles []profile.Profile
	// Index of the currently selected profile.
	profileSelected int
	// Index of the currently selected profile's scroll position.
	breakdownScroll int
}

// NewModel creates a new model.
func NewModel() Model {
	return Model{}
}
