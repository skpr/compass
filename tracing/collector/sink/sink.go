// Package sink is used to declare the interface used for sinks.
package sink

import (
	"context"

	"github.com/skpr/compass/tracing/trace"
)

// Interface for handling profile data.
type Interface interface {
	// Initialize the plugin.
	Initialize() error
	// ProcessTrace which has been collected.
	ProcessTrace(context.Context, trace.Trace) error
}
