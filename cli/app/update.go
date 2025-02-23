package app

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/skpr/compass/cli/app/types"
)

// Update triggers on messages and updates the model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// Handle key presses.
	case tea.KeyMsg:
		switch msg.String() {
		case tea.KeyCtrlC.String():
			return m, tea.Quit

		// For navigating the main menu.
		case tea.KeyRight.String():
			return m.updateKeyRight()
		case tea.KeyLeft.String():
			return m.updateKeyLeft()

		case tea.KeyEnter.String():
			return m.updateKeyEnter()
		}

	case tea.WindowSizeMsg:
		return m.updateWindowSize(msg)

	case types.Trace:
		return m.updateTrace(msg)

	case types.Log:
		return m.updateLog(msg)
	}

	var cmd tea.Cmd

	switch m.PageSelected {
	case types.PageSearch:
		m.search, cmd = m.search.Update(msg)
		return m, cmd
	case types.PageSpans:
		m.spans, cmd = m.spans.Update(msg)
		return m, cmd
	case types.PageTotals:
		m.totals, cmd = m.totals.Update(msg)
		return m, cmd
	case types.PageLogs:
		m.logs, cmd = m.logs.Update(msg)
		return m, cmd
	}

	return m, nil
}
