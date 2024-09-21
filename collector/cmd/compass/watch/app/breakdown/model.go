// Package breakdown for handling the breakdown component.
package breakdown

import "github.com/skpr/compass/collector/cmd/compass/watch/app/profile"

// Model for storing and rendering the state of the breakdown component.
type Model struct {
	Profiles       []profile.Profile
	Selected       int
	ScrollPosition int
}
