package app

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"

	"github.com/skpr/compass/pkg/app/color"
	"github.com/skpr/compass/pkg/app/component/span"
	"github.com/skpr/compass/pkg/trace/segmented"
)

// SpanLength is how long a span component should be.
const SpanLength = 50

func (m *Model) spansInit() {
	m.spans = table.New(
		table.WithFocused(true),
		table.WithHeight(30),
	)

	m.spansSetColumns()

	styles := table.DefaultStyles()

	styles.Header = styles.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(color.White)).
		BorderBottom(true).
		Bold(true)

	styles.Selected = styles.Selected.
		Foreground(lipgloss.Color(color.White)).
		Background(lipgloss.Color(color.DarkBlueGrey)).
		Bold(true)

	m.spans.SetStyles(styles)
}

func (m *Model) spansSetColumns() {
	memory := table.Column{
		Title: "Memory (Inc)",
		Width: 15,
	}

	spans := table.Column{
		Title: "Spans",
		Width: SpanLength + 35,
	}

	functions := table.Column{
		Title: "Functions",
		Width: m.Width - spans.Width - memory.Width + 15,
	}

	m.spans.SetColumns([]table.Column{
		functions,
		memory,
		spans,
	})
}

func (m *Model) spansSetRows() {
	if m.Current == nil {
		return
	}

	trace := segmented.Unmarshal(m.Current.Trace, SpanLength)

	sc := span.New(time.Duration(trace.Metadata.ExecutionTime())*time.Nanosecond, float64(SpanLength))

	var rows []table.Row

	for _, s := range trace.Spans {
		rows = append(rows, []string{
			s.Name,
			formatBytes(s.MaxMemory),
			sc.Render(span.Span{
				Start:    time.Duration(s.Start) * time.Nanosecond,
				Duration: time.Duration(s.Length) * time.Nanosecond,
			}),
		})
	}

	m.spans.SetRows(rows)
}

func (m *Model) spansView() string {
	return m.spans.View()
}

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
