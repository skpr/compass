package app

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/skpr/compass/cli/app/types"
)

func (m *Model) updateKeyLeft() (tea.Model, tea.Cmd) {
	switch m.PageSelected {
	case types.PageLogs:
		m.PageSelected = types.PageTotals
	case types.PageTotals:
		m.PageSelected = types.PageSpans
	case types.PageSpans:
		m.PageSelected = types.PageSearch
	}

	return m, nil
}
