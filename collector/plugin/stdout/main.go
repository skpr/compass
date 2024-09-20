// Package stdout implements a simple plugin that prints to stdout in json format.
package main

import (
	"encoding/json"
	"os"

	"github.com/skpr/compass/collector/pkg/tracing"
)

// Plugin that is exported for use by the collector.
var Plugin plugin

// Plugin for handling profile data.
type plugin struct{}

// Initialize the plugin.
func (s *plugin) Initialize() error {
	return nil
}

// ProcessProfile from the collector.
func (s *plugin) ProcessProfile(profile tracing.Profile) error {
	return json.NewEncoder(os.Stdout).Encode(profile)
}
