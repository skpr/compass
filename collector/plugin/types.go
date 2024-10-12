package plugin

import (
	"github.com/skpr/compass/collector/pkg/tracing/complete"
)

// Interface for handling profile data.
type Interface interface {
	// Initialize the plugin.
	Initialize() error
	// ProcessProfile which has been collected.
	ProcessProfile(complete.Profile) error
}
