package plugin

import "github.com/skpr/compass/collector/internal/event/types"

type Interface interface {
	// Initialize the plugin.
	Initialize() error
	// TraceEnd event which occurs once a trace has been completed and ready to be processed.
	TraceEnd(types.Trace) error
}
