package app

import tea "github.com/charmbracelet/bubbletea"

// ScrollUp when the up arrow key is pressed.
func (m *Model) ScrollUp() (tea.Model, tea.Cmd) {
	if m.breakdownScroll > 0 {
		m.breakdownScroll--
	}

	return m, nil
}
