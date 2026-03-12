package app

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/skpr/compass/pkg/app/events"
)

func (m *Model) updateTrace(trace events.Trace) (tea.Model, tea.Cmd) {
	m.search.InsertItem(len(m.search.Items()), trace)
	m.Traces[trace.Metadata.ID] = trace
	return m, nil
}
