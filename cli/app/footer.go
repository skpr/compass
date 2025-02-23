package app

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"

	"github.com/skpr/compass/cli/app/color"
)

var (
	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: color.White, Dark: color.White}).
			Background(lipgloss.AdaptiveColor{Light: color.Grey, Dark: color.Grey})

	statusNugget = lipgloss.NewStyle().
			Foreground(lipgloss.Color(color.White)).
			Padding(0, 1)

	logoStyle = statusNugget.Background(lipgloss.Color(color.Blue))

	statusText = lipgloss.NewStyle().Inherit(statusBarStyle)
)

func (m *Model) footerView() string {
	logo := logoStyle.Render("Compass")

	status := statusText.
		Width(m.Width - lipgloss.Width(logo)).
		Render(fmt.Sprintf("Using probes from %s", m.ProbePath))

	bar := lipgloss.JoinHorizontal(lipgloss.Top,
		status,
		logo,
	)

	return statusBarStyle.Width(m.Width).Render(bar)
}
