package app

// Model for storing the state of the application.
type Model struct {
	// This is how we track the height and width of the terminal.
	width, height int
	// Internal storage for profiles.
	profiles []Profile
	// Index of the currently selected profile.
	selectedProfile int
	// Index of the currently selected profile's scroll position.
	profileScroll int
}

// NewModel creates a new model.
func NewModel() Model {
	return Model{}
}
