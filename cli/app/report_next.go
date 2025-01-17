package app

import tea "github.com/charmbracelet/bubbletea"

// ReportNext when the shift+right arrow key is pressed.
func (m *Model) ReportNext() (tea.Model, tea.Cmd) {
	m.viewMode = ViewCount
	return m, nil
}
