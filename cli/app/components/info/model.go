// Package info for handling the info component.
package info

import (
	"github.com/skpr/compass/trace"
)

// Model for storing and rendering the state of the list component.
type Model struct {
	Trace    trace.Trace
	Selected int
}
