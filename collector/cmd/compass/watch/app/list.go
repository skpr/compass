package app

import (
	"github.com/charmbracelet/lipgloss"
)

// Helper function for listing all profiles.
func (m Model) profileList() string {
	if len(m.profiles) == 0 {
		return "No profiles available"
	}

	var p []string

	// 4 is the number of lines used to print the summary information.
	start, end := getListStartAndEnd(m.selectedProfile, 4, len(m.profiles))

	for i, profile := range m.profiles {
		if i < start {
			continue
		}

		if i >= end {
			continue
		}

		p = append(p, profileSummary(profile, i == m.selectedProfile))
	}

	// We use 2 lines so we can separate the profiles.
	return lipgloss.JoinHorizontal(lipgloss.Top, p...)
}

// Helper function which returns the start and end of the list that should be displayed.
func getListStartAndEnd(position, visible, length int) (int, int) {
	if length <= visible {
		return 0, length
	}

	if position < visible {
		return 0, visible
	}

	if position+1 > length {
		return length - visible, length
	}

	return position - visible + 1, position + 1
}
