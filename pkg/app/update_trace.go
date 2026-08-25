package app

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/skpr/compass/pkg/app/events"
)

func (m *Model) updateTrace(trace events.Trace) (tea.Model, tea.Cmd) {
	// Newest traces are shown first so the most recent activity does not
	// require scrolling to the bottom of the list.
	m.traces = append([]events.Trace{trace}, m.traces...)

	m.evictTraces()
	m.searchSetRows()

	return m, nil
}

// evictTraces discards the oldest traces so that we retain at most MaxTraces.
//
// Traces arrive for as long as the CLI is open, so without a cap both the list
// and the memory it holds grow for the life of the session.
func (m *Model) evictTraces() {
	limit := m.MaxTraces
	if limit <= 0 {
		limit = DefaultMaxTraces
	}

	if len(m.traces) > limit {
		m.traces = m.traces[:limit]
	}
}
