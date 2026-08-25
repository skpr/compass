package app

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/skpr/compass/pkg/app/events"
)

func (m *Model) updateLog(log events.Log) (tea.Model, tea.Cmd) {
	// Newest first, matching the trace list.
	m.logs = append([]events.Log{log}, m.logs...)

	m.logsSetRows()

	return m, nil
}
