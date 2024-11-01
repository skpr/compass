// Package breakdown for handling the breakdown component.
package breakdown

import (
	"github.com/skpr/compass/profile/complete"
)

// Model for storing and rendering the state of the breakdown component.
type Model struct {
	Profiles       []complete.Profile
	Selected       int
	ScrollPosition int
}