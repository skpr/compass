// Package noop implements a no-op plugin for the collector.
package main

import (
	"github.com/skpr/compass/collector/pkg/tracing/complete"
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
func (s *plugin) ProcessProfile(_ complete.Profile) error {
	return nil
}
