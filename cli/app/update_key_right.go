package app

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/skpr/compass/cli/app/types"
)

func (m *Model) updateKeyRight() (tea.Model, tea.Cmd) {
	switch m.PageSelected {
	case types.PageSearch:
		m.PageSelected = types.PageSpans
	case types.PageSpans:
		m.PageSelected = types.PageTotals
	case types.PageTotals:
		m.PageSelected = types.PageLogs
	}

	return m, nil
}
