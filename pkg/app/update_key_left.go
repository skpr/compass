package app

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) updateKeyLeft() (tea.Model, tea.Cmd) {
	switch m.PageSelected {
	case PageLogs:
		m.selectPage(PageSearch)
	case PageDrupal:
		m.selectPage(PageFunctions)
	}

	m.relayout()

	return m, nil
}
