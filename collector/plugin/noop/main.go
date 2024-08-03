package main

import (
	"github.com/skpr/compass/collector/internal/event/types"
)

// Plugin that is exported for use by the collector.
var Plugin plugin

type plugin struct{}

func (s *plugin) Initialize() error {
	return nil
}

// ProcessRequest from the collector.
func (s *plugin) ProcessRequest(request types.Request) error {
	return nil
}
