package app

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/skpr/compass/pkg/app/events"
)

func (m *Model) updateSummaryResult(msg events.SummaryResult) (tea.Model, tea.Cmd) {
	m.summaryCache[msg.TraceID] = msg.Text
	m.summaryLoading = false
	m.summaryError = ""

	// Only update the displayed text if we're still looking at the same trace.
	if m.Current != nil && m.Current.Metadata.ID == msg.TraceID {
		m.summaryText = msg.Text
	}

	return m, nil
}

func (m *Model) updateSummaryError(msg events.SummaryError) (tea.Model, tea.Cmd) {
	m.summaryLoading = false

	// Only update the displayed error if we're still looking at the same trace.
	if m.Current != nil && m.Current.Metadata.ID == msg.TraceID {
		m.summaryError = msg.Err.Error()
	}

	return m, nil
}
