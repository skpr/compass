package app

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"

	"github.com/skpr/compass/pkg/app/color"
)

var (
	footerBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: color.White, Dark: color.White}).
			Background(lipgloss.AdaptiveColor{Light: color.Grey, Dark: color.Grey})

	logo = lipgloss.NewStyle().
		Foreground(lipgloss.Color(color.White)).
		Padding(0, 1).Background(lipgloss.Color(color.Blue)).
		Render("Compass")

	statusText = lipgloss.NewStyle().Inherit(footerBarStyle)
)

func (m *Model) footerView() string {
	var statusContent string

	if m.PageSelected == PageSpans && m.Current != nil {
		statusContent = fmt.Sprintf("Using probes from %s  |  [s] AI Summary", m.ProbePath)
	} else {
		statusContent = fmt.Sprintf("Using probes from %s", m.ProbePath)
	}

	status := statusText.
		Width(m.Width - lipgloss.Width(logo)).
		Render(statusContent)

	bar := lipgloss.JoinHorizontal(lipgloss.Top,
		status,
		logo,
	)

	return footerBarStyle.Width(m.Width).Render(bar)
}
