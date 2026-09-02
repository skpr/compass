package app

import (
	tea "github.com/charmbracelet/bubbletea"
)

// updateKeyEsc closes whatever is open: the help overlay first, then the trace.
func (m *Model) updateKeyEsc() (tea.Model, tea.Cmd) {
	if m.showHelp {
		m.showHelp = false

		return m, nil
	}

	m.selectPage(PageSearch)
	m.relayout()

	return m, nil
}
