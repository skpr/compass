package app

import tea "github.com/charmbracelet/bubbletea"

// ReportPrevious when the shift+left arrow key is pressed.
func (m *Model) ReportPrevious() (tea.Model, tea.Cmd) {
	m.viewMode = ViewSpans
	return m, nil
}
