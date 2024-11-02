// Package sink implements a simple sink that prints to stdout.
package sink

import (
	"os"

	"encoding/json"

	"github.com/skpr/compass/profile/aggregated"
	"github.com/skpr/compass/profile/complete"
)

// New client for handling profiles to stdout.
func New(functionThreshold, requestThreshold int64) *Client {
	return &Client{
		functionThreshold: functionThreshold,
		requestThreshold:  requestThreshold,
	}
}

// Client for handling profiles to stdout.
type Client struct {
	functionThreshold int64
	requestThreshold  int64
}

// Initialize the plugin.
func (c *Client) Initialize() error {
	return nil
}

// ProcessProfile from the collector.
func (c *Client) ProcessProfile(completeProfile complete.Profile) error {
	if completeProfile.ExecutionTime < c.requestThreshold {
		return nil
	}

	profile := aggregated.FromCompleteProfile(completeProfile)

	for name, function := range profile.Functions {
		if function.ExecutionTime < c.functionThreshold {
			delete(profile.Functions, name)
		}
	}

	return json.NewEncoder(os.Stdout).Encode(completeProfile)
}
