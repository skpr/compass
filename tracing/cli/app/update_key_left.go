package app

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) updateKeyLeft() (tea.Model, tea.Cmd) {
	switch m.PageSelected {
	case PageLogs:
		m.PageSelected = PageTotals
	case PageTotals:
		m.PageSelected = PageSpans
	case PageSpans:
		m.PageSelected = PageSearch
	}

	return m, nil
}
