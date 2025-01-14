package app

import tea "github.com/charmbracelet/bubbletea"

// TracePrevious when the left arrow key is pressed.
func (m *Model) TracePrevious() (tea.Model, tea.Cmd) {
	if m.traceSelected > 0 {
		m.traceSelected--
		m.breakdownScroll = 0
	}

	return m, nil
}
