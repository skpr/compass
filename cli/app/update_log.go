package app

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/skpr/compass/cli/app/types"
)

func (m *Model) updateLog(log types.Log) (tea.Model, tea.Cmd) {
	m.logs.InsertItem(0, log)
	return m, nil
}
