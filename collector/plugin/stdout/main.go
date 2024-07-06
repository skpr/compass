package main

import (
	"encoding/json"
	"os"

	"github.com/skpr/compass/collector/internal/event/types"
)

// Plugin that is exported for use by the collector.
var Plugin plugin

type plugin struct{}

func (s *plugin) Initialize() error {
	return nil
}

// TraceEnd event from the collector.
func (s *plugin) TraceEnd(trace types.Trace) error {
	return json.NewEncoder(os.Stdout).Encode(trace)
}
