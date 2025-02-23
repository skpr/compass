package app

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/skpr/compass/cli/app/types"
)

// Init initializes the model.
func (m *Model) Init() tea.Cmd {
	m.PageSelected = types.PageSearch

	m.Traces = make(map[string]types.Trace)

	m.searchInit()
	m.logsInit()
	m.metadataInit()
	m.spansInit()
	m.totalsInit()

	return nil
}
