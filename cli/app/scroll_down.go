package app

import (
	tea "github.com/charmbracelet/bubbletea"

	compasstable "github.com/skpr/compass/cli/app/components/table"
)

// ScrollDown when the up down key is pressed.
func (m *Model) ScrollDown() (tea.Model, tea.Cmd) {
	if m.breakdownScroll+compasstable.VisibleRows < len(m.traces.Filtered[m.traceSelected].FunctionCalls)-1 {
		m.breakdownScroll++
	}

	return m, nil
}
