package plugin

import "github.com/skpr/compass/collector/internal/event/types"

type Interface interface {
	// Initialize the plugin.
	Initialize() error
	// ProcessProfile which has been collected.
	ProcessProfile(types.Profile) error
}
