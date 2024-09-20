// Package cache implements a plugin for storing profiles to a common cache.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/skpr/compass/collector/pkg/tracing"
)

const (
	// EnvEndpoint to send profile events to.
	EnvEndpoint = "COMPASS_COLLECTOR_CLI_ENDPOINT"
	// DefaultEndpoint for sending profile events.
	DefaultEndpoint = "http://localhost:27624"
)

// Plugin that is exported for use by the collector.
var Plugin plugin

// Plugin for handling profile data.
type plugin struct {
	endpoint string
}

// Initialize the plugin.
func (s *plugin) Initialize() error {
	s.endpoint = os.Getenv(EnvEndpoint)

	if s.endpoint == "" {
		s.endpoint = DefaultEndpoint
	}

	return nil
}

// ProcessProfile from the collector.
func (s *plugin) ProcessProfile(profile tracing.Profile) error {
	if profile.RequestID == "" {
		return fmt.Errorf("not found: request id")
	}

	body, err := json.Marshal(profile)
	if err != nil {
		return fmt.Errorf("failed to marshal trace data to json: %w", err)
	}

	_, err = http.Post(s.endpoint, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to send profile event: %w", err)
	}

	return nil
}
