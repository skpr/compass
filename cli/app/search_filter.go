package app

import (
	"strings"

	"github.com/skpr/compass/trace"
)

// SearchFilter all traces.
func (m *Model) SearchFilter() {
	if m.searchQuery == "" {
		m.traces.Filtered = m.traces.All
		return
	}

	var filtered []trace.Trace

	for _, trace := range m.traces.All {
		if strings.Contains(trace.Metadata.URI, m.searchQuery) {
			filtered = append(filtered, trace)
		}
	}

	m.traces.Filtered = filtered
}
