// Package list for handling the list component.
package list

import (
	"github.com/skpr/compass/collector/cmd/compass/watch/app/profile"
)

// Model for storing and rendering the state of the list component.
type Model struct {
	Profiles        []profile.Profile
	Selected        int
	VisibleProfiles int
	Height          int
	Width           int
}
