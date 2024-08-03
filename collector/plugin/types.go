package plugin

import (
	"github.com/skpr/compass/collector/internal/tracing"
)

// Interface for handling profile data.
type Interface interface {
	// Initialize the plugin.
	Initialize() error
	// ProcessProfile which has been collected.
	ProcessProfile(tracing.Profile) error
}
