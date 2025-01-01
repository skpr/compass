package app

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/skpr/compass/cli/app/components/view/spans"
	"github.com/skpr/compass/trace"
)

// Update triggers on messages and updates the model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// Handle key presses.
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit

		// For switching between profiles.
		case "left":
			if m.traceSelected > 0 {
				m.traceSelected--
				m.breakdownScroll = 0
			}
		case "right":
			if m.traceSelected < len(m.traces)-1 {
				m.traceSelected++
				m.breakdownScroll = 0
			}

		case "s":
			m.viewMode = ViewSpans
		case "d":
			m.viewMode = ViewDescription

		// For scrolling the profile.
		case "up":
			if m.breakdownScroll > 0 {
				m.breakdownScroll--
			}
		case "down":
			if len(m.traces) <= m.traceSelected {
				return m, nil
			}

			if m.breakdownScroll+spans.VisibleRows < len(m.traces[m.traceSelected].FunctionCalls)-1 {
				m.breakdownScroll++
			}
		}

	// When a new profile is received, add it to the list of profiles.
	case trace.Trace:
		m.traces = append(m.traces, msg)
		return m, nil

	// Window was resized.
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	}

	return m, nil
}
