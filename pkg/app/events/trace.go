package events

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/skpr/compass/pkg/app/color"
	skprtime "github.com/skpr/compass/pkg/app/time"
	"github.com/skpr/compass/pkg/trace"
)

// Column styles for runtime and source badges.
var (
	runtimeStyles = map[trace.Runtime]lipgloss.Style{
		trace.RuntimePHP:  lipgloss.NewStyle().Width(6).Foreground(lipgloss.Color(color.Blue)),
		trace.RuntimeNode: lipgloss.NewStyle().Width(6).Foreground(lipgloss.Color(color.Green)),
	}

	sourceStyles = map[trace.Source]lipgloss.Style{
		trace.SourceHTTP: lipgloss.NewStyle().Width(6).Foreground(lipgloss.Color(color.Yellow)),
		trace.SourceCLI:  lipgloss.NewStyle().Width(6).Foreground(lipgloss.Color(color.Orange)),
	}

	defaultBadgeStyle = lipgloss.NewStyle().Width(6).Foreground(lipgloss.Color(color.Grey))

	// Duration severity styles.
	durationStyleFast     = lipgloss.NewStyle().Width(8).Foreground(lipgloss.Color(color.Green))
	durationStyleModerate = lipgloss.NewStyle().Width(8).Foreground(lipgloss.Color(color.Yellow))
	durationStyleSlow     = lipgloss.NewStyle().Width(8).Foreground(lipgloss.Color(color.Red))

	// Memory column style.
	memoryStyle = lipgloss.NewStyle().Width(10).Foreground(lipgloss.Color(color.Grey))

	// Bold style for HTTP method and CLI command.
	boldStyle = lipgloss.NewStyle().Bold(true)

	// Description style.
	descDimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(color.Grey))
)

const (
	// Duration thresholds in milliseconds.
	durationThresholdModerate = 100
	durationThresholdSlow     = 500

	// Duration bar settings.
	durationBarWidth  = 8
	durationBarMaxMs  = 1000
	durationBarFilled = '█'
	durationBarEmpty  = '░'
)

// Trace for review.
type Trace struct {
	IngestionTime time.Time
	trace.Trace
}

// runtimeBadge returns a fixed-width, color-coded runtime label.
func (t Trace) runtimeBadge() string {
	if style, ok := runtimeStyles[t.Metadata.Runtime]; ok {
		return style.Render(string(t.Metadata.Runtime))
	}

	return defaultBadgeStyle.Render(string(t.Metadata.Runtime))
}

// sourceBadge returns a fixed-width, color-coded source label.
func (t Trace) sourceBadge() string {
	if style, ok := sourceStyles[t.Metadata.Source]; ok {
		return style.Render(string(t.Metadata.Source))
	}

	return defaultBadgeStyle.Render(string(t.Metadata.Source))
}

// durationText returns a fixed-width, severity-colored duration string.
func (t Trace) durationText() string {
	ms := skprtime.NanosecondsToMilliseconds(t.Metadata.ExecutionTime())
	text := fmt.Sprintf("%dms", ms)

	switch {
	case ms >= durationThresholdSlow:
		return durationStyleSlow.Render(text)
	case ms >= durationThresholdModerate:
		return durationStyleModerate.Render(text)
	default:
		return durationStyleFast.Render(text)
	}
}

// durationBar returns an 8-character sparkline bar colored by severity.
// The bar scales linearly from 0 to 1000ms, clamped at both ends.
func (t Trace) durationBar() string {
	ms := skprtime.NanosecondsToMilliseconds(t.Metadata.ExecutionTime())

	filled := int((ms * durationBarWidth) / durationBarMaxMs)
	if filled < 1 && ms > 0 {
		filled = 1
	}

	if filled > durationBarWidth {
		filled = durationBarWidth
	}

	bar := make([]rune, durationBarWidth)
	for i := range bar {
		if i < filled {
			bar[i] = durationBarFilled
		} else {
			bar[i] = durationBarEmpty
		}
	}

	text := string(bar)

	switch {
	case ms >= durationThresholdSlow:
		return lipgloss.NewStyle().Foreground(lipgloss.Color(color.Red)).Render(text)
	case ms >= durationThresholdModerate:
		return lipgloss.NewStyle().Foreground(lipgloss.Color(color.Yellow)).Render(text)
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color(color.Green)).Render(text)
	}
}

// memoryText returns a fixed-width, dimmed memory string.
func (t Trace) memoryText() string {
	return memoryStyle.Render(formatBytes(t.ResourceUtilisation.MaxMemory))
}

// Title of the trace.
func (t Trace) Title() string {
	runtime := t.runtimeBadge()
	source := t.sourceBadge()
	duration := t.durationText()
	bar := t.durationBar()
	memory := t.memoryText()

	var identity string

	switch t.Metadata.Source {
	case trace.SourceHTTP:
		identity = fmt.Sprintf("%s %s", boldStyle.Render(t.Metadata.HTTP.Method), t.Metadata.HTTP.URI)
	case trace.SourceCLI:
		identity = boldStyle.Render(t.Metadata.CLI.Command)
	default:
		identity = "UNKNOWN"
	}

	return fmt.Sprintf("%s %s %s %s %s %s", runtime, source, duration, bar, memory, identity)
}

// Description of the trace.
func (t Trace) Description() string {
	count := fmt.Sprintf("%d calls", len(t.FunctionCalls))
	ago := relativeTime(t.IngestionTime)

	return descDimStyle.Render(fmt.Sprintf("id=%s, %s, %s", t.Metadata.ID, count, ago))
}

// FilterValue for search.
func (t Trace) FilterValue() string {
	var identity string

	switch t.Metadata.Source {
	case trace.SourceHTTP:
		identity = fmt.Sprintf("%s %s", t.Metadata.HTTP.Method, t.Metadata.HTTP.URI)
	case trace.SourceCLI:
		identity = t.Metadata.CLI.Command
	default:
		identity = "UNKNOWN"
	}

	return fmt.Sprintf("%s %s %s", t.Metadata.Runtime, t.Metadata.Source, identity)
}

// relativeTime returns a human-readable relative time string.
func relativeTime(t time.Time) string {
	if t.IsZero() {
		return "just now"
	}

	d := time.Since(t)

	switch {
	case d < time.Second:
		return "just now"
	case d < time.Minute:
		secs := int(d.Seconds())
		if secs == 1 {
			return "1s ago"
		}

		return fmt.Sprintf("%ds ago", secs)
	case d < time.Hour:
		mins := int(d.Minutes())
		if mins == 1 {
			return "1m ago"
		}

		return fmt.Sprintf("%dm ago", mins)
	default:
		hours := int(d.Hours())
		if hours == 1 {
			return "1h ago"
		}

		return fmt.Sprintf("%dh ago", hours)
	}
}

// formatBytes returns the bytes in a human readable format.
func formatBytes(b int64) string {
	const unit = 1024

	if b < unit {
		return fmt.Sprintf("%d B", b)
	}

	div, exp := int64(unit), 0

	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
