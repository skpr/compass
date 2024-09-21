// Package stdout implements a simple plugin that prints to stdout in json format.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/skpr/compass/collector/pkg/tracing/aggregated"
	"github.com/skpr/compass/collector/pkg/tracing/complete"
)

const (
	// EnvRequestThreshold is used to configure the request threshold.
	EnvRequestThreshold = "COMPASS_COLLECTOR_STDOUT_REQUEST_THRESHOLD"
	// EnvFunctionThreshold is used to configure the function threshold.
	EnvFunctionThreshold = "COMPASS_COLLECTOR_STDOUT_FUNCTION_THRESHOLD"

	// DefaultRequestThreshold for the plugin.
	DefaultRequestThreshold = 10000
	// DefaultFunctionThreshold for the plugin.
	DefaultFunctionThreshold = 1000
)

// Plugin that is exported for use by the collector.
var Plugin plugin

// Plugin for handling profile data.
type plugin struct {
	requestThreshold  int64
	functionThreshold int64
}

// Initialize the plugin.
func (s *plugin) Initialize() error {
	s.requestThreshold = DefaultRequestThreshold
	s.functionThreshold = DefaultFunctionThreshold

	var (
		requestThreshold  = os.Getenv(EnvRequestThreshold)
		functionThreshold = os.Getenv(EnvFunctionThreshold)
	)

	if requestThreshold != "" {
		rt, err := strconv.ParseInt(requestThreshold, 10, 64)
		if err != nil {
			return fmt.Errorf("failed to parse request threshold: %w", err)
		}

		s.requestThreshold = rt
	}

	if functionThreshold != "" {
		rt, err := strconv.ParseInt(functionThreshold, 10, 64)
		if err != nil {
			return fmt.Errorf("failed to parse function threshold: %w", err)
		}

		s.functionThreshold = rt
	}

	return nil
}

// ProcessProfile from the collector.
func (s *plugin) ProcessProfile(profile complete.Profile) error {
	/*if profile.ExecutionTime < s.requestThreshold {
		return nil
	}*/

	return json.NewEncoder(os.Stdout).Encode(aggregated.FromCompleteProfile(profile))
}
