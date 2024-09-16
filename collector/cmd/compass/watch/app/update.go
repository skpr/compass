package app

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/skpr/compass/collector/pkg/tracing"
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
			if m.selectedProfile > 0 {
				m.selectedProfile--
				m.profileScroll = 0
			}
		case "right":
			if m.selectedProfile < len(m.profiles)-1 {
				m.selectedProfile++
				m.profileScroll = 0
			}

		// For scrolling the profile.
		case "up":
			if m.profileScroll > 0 {
				m.profileScroll--
			}
		case "down":
			if len(m.profiles) < m.selectedProfile {
				return m, nil
			}

			if m.profileScroll+TableRows < len(m.profiles[m.selectedProfile].Functions)-1 {
				m.profileScroll++
			}
		}

	// When a new profile is received, add it to the list of profiles.
	case tracing.Profile:
		m.profiles = append(m.profiles, WrapProfile(msg))
		return m, nil

	// Window was resized.
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	}

	return m, nil
}
