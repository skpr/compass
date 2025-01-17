package app

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/skpr/compass/trace"
)

// Update triggers on messages and updates the model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// Handle key presses.
	case tea.KeyMsg:
		switch msg.String() {
		case tea.KeyCtrlC.String():
			return m, tea.Quit

		// For moving between traces.
		case tea.KeyLeft.String():
			return m.TracePrevious()
		case tea.KeyRight.String():
			return m.TraceNext()

		// For scrolling the trace details.
		case tea.KeyUp.String():
			return m.ScrollUp()
		case tea.KeyDown.String():
			return m.ScrollDown()

		// Allow views to tab through analysis reports.
		case tea.KeyShiftLeft.String():
			return m.ReportPrevious()
		case tea.KeyShiftRight.String():
			return m.ReportNext()

		// Delete character from search.
		case tea.KeyBackspace.String():
			return m.SearchQueryDelete()

		// Add character to search.
		default:
			return m.SearchQueryAdd(msg)
		}

	// When a new profile is received, add it to the list of profiles.
	case trace.Trace:
		m.traces.All = append(m.traces.All, msg)
		m.SearchFilter()
		return m, nil

	// Window was resized.
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	}

	return m, nil
}
