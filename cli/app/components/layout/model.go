// Package layout for handling the layout component.
package layout

import (
	"github.com/skpr/compass/trace"
)

// Component for rendering the layout.
type Component interface {
	View() string
}

// Model for storing and rendering the state of the layout.
type Model struct {
	Info      Component
	Breakdown Component
	Help      Component
	Traces    []trace.Trace
	Selected  int
}
