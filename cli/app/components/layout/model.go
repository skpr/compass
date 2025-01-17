// Package layout for handling the layout component.
package layout

// Component for rendering the layout.
type Component interface {
	View() string
}

// Model for storing and rendering the state of the layout.
type Model struct {
	Search   Component
	Info     Component
	Spans    Component
	Help     Component
	Total    int
	Selected int
}
