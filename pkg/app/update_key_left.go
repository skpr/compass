package app

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) updateKeyLeft() (tea.Model, tea.Cmd) {
	switch m.PageSelected {
	case PageLogs:
		m.PageSelected = PageSearch
	case PageDrupal:
		m.PageSelected = PageFunctions
	}

	m.relayout()

	return m, nil
}
