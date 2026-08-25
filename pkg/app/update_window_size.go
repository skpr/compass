package app

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) updateWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.Height = msg.Height
	m.Width = msg.Width

	m.relayout()

	// The rows carry their own widths, so a resize rebuilds them rather than
	// leaving the previous terminal's arithmetic behind.
	m.searchSetRows()
	m.logsSetRows()
	m.functionsSetRows()
	m.drupalSetRows()

	return m, nil
}
