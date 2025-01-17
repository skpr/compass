// Package count for visualising the trace as a set total counts.
package count

import (
	"github.com/skpr/compass/trace"
)

// Model for storing and rendering the state of the breakdown component.
type Model struct {
	Trace          trace.Trace
	ScrollPosition int
}
