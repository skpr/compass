// Package breakdown for handling the breakdown component.
package breakdown

import (
	"github.com/skpr/compass/trace"
)

// Model for storing and rendering the state of the breakdown component.
type Model struct {
	Traces         []trace.Trace
	Selected       int
	ScrollPosition int
}
