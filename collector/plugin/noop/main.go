package main

import (
	"github.com/skpr/compass/collector/internal/tracing"
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
	return nil
}
