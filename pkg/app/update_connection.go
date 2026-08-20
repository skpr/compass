package app

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/skpr/compass/pkg/app/events"
)

func (m *Model) updateConnection(msg events.Connection) (tea.Model, tea.Cmd) {
	m.connection = msg
	return m, nil
}
