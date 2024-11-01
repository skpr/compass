package app

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/skpr/compass/cli/app/components/breakdown"
	"github.com/skpr/compass/profile/complete"
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
			if m.profileSelected > 0 {
				m.profileSelected--
				m.breakdownScroll = 0
			}
		case "right":
			if m.profileSelected < len(m.profiles)-1 {
				m.profileSelected++
				m.breakdownScroll = 0
			}

		// For scrolling the profile.
		case "up":
			if m.breakdownScroll > 0 {
				m.breakdownScroll--
			}
		case "down":
			if len(m.profiles) < m.profileSelected {
				return m, nil
			}

			if m.breakdownScroll+breakdown.VisibleRows < len(m.profiles[m.profileSelected].FunctionCalls)-1 {
				m.breakdownScroll++
			}
		}

	// When a new profile is received, add it to the list of profiles.
	case complete.Profile:
		m.profiles = append(m.profiles, msg)
		return m, nil

	// Window was resized.
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	}

	return m, nil
}
