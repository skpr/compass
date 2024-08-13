// Package cache implements a plugin for storing profiles to a common cache.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/skpr/compass/collector/internal/tracing"
)

const (
	// EnvDirectory is used to configure this plugin.
	EnvDirectory = "COMPASS_COLLECTOR_PLUGIN_CACHE_DIRECTORY"
)

// Plugin that is exported for use by the collector.
var Plugin plugin

// Plugin for handling profile data.
type plugin struct {
	directory string
}

// Initialize the plugin.
func (s *plugin) Initialize() error {
	directory := os.Getenv(EnvDirectory)

	// Return early if a specific directory has been provided.
	if directory != "" {
		s.directory = directory
		return nil
	}

	// Fallback to users cache directory.
	base, err := os.UserCacheDir()
	if err != nil {
		return fmt.Errorf("failed to get cache directory: %w", err)
	}

	s.directory = fmt.Sprintf("%s/%s", base, "compass")

	return nil
}

// ProcessProfile from the collector.
func (s *plugin) ProcessProfile(profile tracing.Profile) error {
	if profile.RequestID == "" {
		return fmt.Errorf("not found: request id")
	}

	contents, err := json.Marshal(profile)
	if err != nil {
		return fmt.Errorf("failed to marshal trace data to json: %w", err)
	}

	file := fmt.Sprintf("%s/%s.json", s.directory, profile.RequestID)

	err = os.WriteFile(file, contents, 0644)
	if err != nil {
		return fmt.Errorf("failed to write trace data to file: %w", err)
	}

	return nil
}
