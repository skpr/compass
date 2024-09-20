package list

import (
	"github.com/charmbracelet/lipgloss"
)

// Lists all available profiles and highlights the selected profile.
func (m Model) getList() string {
	if len(m.Profiles) == 0 {
		return "No profiles available"
	}

	var p []string

	// 4 is the number of lines used to print the summary information.
	start, end := getPositionStartAndEnd(m.Selected, m.VisibleProfiles, len(m.Profiles))

	for i, item := range m.Profiles {
		if i < start {
			continue
		}

		if i >= end {
			continue
		}

		p = append(p, profileSummary(item, i == m.Selected))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, p...)
}
