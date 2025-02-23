package app

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) updateWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.Height = msg.Height
	m.Width = msg.Width

	// Search.
	m.search.SetSize(msg.Width, msg.Height-5)

	// Metadata (on the Spans page)
	m.metadataSetColums()

	// Spans.
	m.spans.SetHeight(msg.Height - 14)
	m.spans.SetWidth(msg.Width)
	m.spansSetColumns()
	m.spansSetRows()
	m.totals.SetHeight(msg.Height - 14)
	m.totals.SetWidth(msg.Width)
	m.totalsSetColumns()
	m.totalsSetRows()

	// Log events.
	m.logs.SetSize(msg.Width, msg.Height-5)

	return m, nil
}
