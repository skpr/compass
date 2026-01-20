package time

import "time"

// Client for interacting with time.
type Client struct{}

// New client for interacting with time.
func New() Client {
	return Client{}
}

// Now returns the current time.
func (client Client) Now() time.Time {
	return time.Now()
}
