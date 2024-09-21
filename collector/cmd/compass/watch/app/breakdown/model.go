// Package breakdown for handling the breakdown component.
package breakdown

import (
	"github.com/skpr/compass/collector/pkg/tracing/aggregated"
)

// Model for storing and rendering the state of the breakdown component.
type Model struct {
	Profiles       []aggregated.Profile
	Selected       int
	ScrollPosition int
}
