package app

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/skpr/compass/pkg/app/color"
	"github.com/skpr/compass/pkg/app/events"
	"github.com/skpr/compass/pkg/ollama"
	"github.com/skpr/compass/pkg/trace"
)

// fetchSummary returns a Bubble Tea command that calls Ollama to generate a trace summary.
func fetchSummary(ollamaURL, ollamaModel string, traceID string, t trace.Trace) tea.Cmd {
	return func() tea.Msg {
		client := ollama.NewClient(ollamaURL, ollamaModel)

		text, err := client.Summarize(t)
		if err != nil {
			return events.SummaryError{
				TraceID: traceID,
				Err:     err,
			}
		}

		return events.SummaryResult{
			TraceID: traceID,
			Text:    text,
		}
	}
}

var (
	overlayStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(color.Blue)).
			Background(lipgloss.Color("#1a1a2e")).
			Foreground(lipgloss.Color(color.White)).
			Padding(1, 2)

	overlayTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color(color.Orange)).
				MarginBottom(1)

	overlayLoadingStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(color.Yellow))

	overlayErrorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(color.Red))

	overlayHintStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(color.Grey)).
				MarginTop(1)
)

// summaryView renders the overlay content for the AI summary.
func (m *Model) summaryView() string {
	overlayWidth := m.Width * 80 / 100
	overlayHeight := m.Height * 70 / 100

	contentWidth := overlayWidth - 6 // Account for border + padding.

	var content strings.Builder

	content.WriteString(overlayTitleStyle.Render("AI Performance Summary"))
	content.WriteString("\n")

	switch {
	case m.summaryLoading:
		content.WriteString(overlayLoadingStyle.Render("Analyzing trace with Ollama..."))
	case m.summaryError != "":
		content.WriteString(overlayErrorStyle.Render("Error: " + m.summaryError))
	default:
		// Word-wrap the summary text to fit the overlay width.
		content.WriteString(wrapText(m.summaryText, contentWidth))
	}

	content.WriteString("\n")
	content.WriteString(overlayHintStyle.Render("Press [s] to close"))

	rendered := overlayStyle.
		Width(overlayWidth).
		Height(overlayHeight).
		Render(content.String())

	return rendered
}

// wrapText wraps text to fit within the given width, preserving existing newlines.
func wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}

	var result strings.Builder

	lines := strings.Split(text, "\n")

	for i, line := range lines {
		if i > 0 {
			result.WriteString("\n")
		}

		if len(line) <= width {
			result.WriteString(line)
			continue
		}

		remaining := line

		for len(remaining) > width {
			// Find the last space before the width limit.
			breakAt := strings.LastIndex(remaining[:width], " ")
			if breakAt <= 0 {
				breakAt = width
			}

			result.WriteString(remaining[:breakAt])
			result.WriteString("\n")

			remaining = strings.TrimLeft(remaining[breakAt:], " ")
		}

		result.WriteString(remaining)
	}

	return result.String()
}
