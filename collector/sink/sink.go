// Package sink is used to declare the interface used for sinks.
package sink

import (
	"github.com/skpr/compass/trace"
)

// Interface for handling profile data.
type Interface interface {
	// Initialize the plugin.
	Initialize() error
	// ProcessTrace which has been collected.
	ProcessTrace(trace.Trace) error
}
