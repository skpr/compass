// Package info for handling the info component.
package info

import (
	"github.com/skpr/compass/collector/pkg/tracing/aggregated"
)

// Model for storing and rendering the state of the list component.
type Model struct {
	Profiles []aggregated.Profile
	Selected int
}
