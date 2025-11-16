package app

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/skpr/compass/tracing/cli/app/events"
)

func (m *Model) updateTrace(trace events.Trace) (tea.Model, tea.Cmd) {
	m.search.InsertItem(len(m.search.Items()), trace)
	m.Traces[trace.Metadata.RequestID] = trace
	return m, nil
}
