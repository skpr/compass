// Package metadata for handling the info component.
package metadata

import (
	"github.com/skpr/compass/profile/complete"
)

// Model for storing and rendering the state of the list component.
type Model struct {
	Profiles []complete.Profile
	Selected int
}
