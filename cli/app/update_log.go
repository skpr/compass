package app

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/skpr/compass/cli/app/events"
)

func (m *Model) updateLog(log events.Log) (tea.Model, tea.Cmd) {
	m.logs.InsertItem(0, log)
	return m, nil
}
