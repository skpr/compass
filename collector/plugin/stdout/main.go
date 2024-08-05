package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/skpr/compass/collector/internal/tracing"
)

const (
	// EnvThreshold is used to set our threshold for filtering out small function calls.
	EnvThreshold = "COMPASS_COLLECTOR_PLUGIN_STDOUT_THRESHOLD"

	// DefaultThreshold for filtering out function calls.
	// This default = 1ms (in nanoseconds)
	DefaultThreshold = 1000000
)

// Plugin that is exported for use by the collector.
var Plugin plugin

// Plugin for handling profile data.
type plugin struct {
	threshold int64
}

// Initialize the plugin.
func (s *plugin) Initialize() error {
	threshold := os.Getenv(EnvThreshold)

	if threshold == "" {
		s.threshold = DefaultThreshold
	}

	parsedThreshold, err := strconv.ParseInt(threshold, 10, 64)
	if err != nil {
		return fmt.Errorf("unable to parse environment variable %s=%s", EnvThreshold, threshold)
	}

	s.threshold = parsedThreshold

	return nil
}

// ProcessProfile from the collector.
func (s *plugin) ProcessProfile(profile tracing.Profile) error {
	return json.NewEncoder(os.Stdout).Encode(reduce(profile, s.threshold))
}

// Helper function to reduce the profile output for stdout and cut out unnecessary noise.
func reduce(profile tracing.Profile, threshold int64) tracing.Profile {
	for name, function := range profile.Functions {
		if function.TotalExecutionTime < uint64(threshold) {
			delete(profile.Functions, name)
		}
	}

	return profile
}
