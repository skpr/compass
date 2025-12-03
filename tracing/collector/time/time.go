package time

import "time"

// Interface for interacting with time.
type Interface interface {
	Now() time.Time
}
