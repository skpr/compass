package app

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"

	"github.com/skpr/compass/pkg/app/color"
	"github.com/skpr/compass/pkg/app/events"
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

	connectionStyles = map[events.ConnectionState]lipgloss.Style{
		events.ConnectionStateConnected:  lipgloss.NewStyle().Inherit(footerBarStyle).Foreground(lipgloss.Color(color.Green)),
		events.ConnectionStateConnecting: lipgloss.NewStyle().Inherit(footerBarStyle).Foreground(lipgloss.Color(color.Yellow)),
		events.ConnectionStateRetrying:   lipgloss.NewStyle().Inherit(footerBarStyle).Foreground(lipgloss.Color(color.Red)),
	}
)

func (m *Model) footerView() string {
	statusContent := fmt.Sprintf("Using probes from %s", m.ProbePath)

	if connection := m.connectionStatus(); connection != "" {
		statusContent = fmt.Sprintf("%s  |  %s", statusContent, connection)
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

// connectionStatus renders the state of the trace stream connection.
func (m *Model) connectionStatus() string {
	if m.connection.State == "" {
		return ""
	}

	text := string(m.connection.State)

	if m.connection.State == events.ConnectionStateRetrying && m.connection.Err != nil {
		text = fmt.Sprintf("%s (%s)", text, m.connection.Err)
	}

	if style, ok := connectionStyles[m.connection.State]; ok {
		return style.Render(text)
	}

	return statusText.Render(text)
}
