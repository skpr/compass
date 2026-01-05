package app

import (
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"

	"github.com/skpr/compass/tracing/cli/app/color"
	"github.com/skpr/compass/tracing/cli/app/component/span"
	"github.com/skpr/compass/tracing/trace/aggregated"
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
		Bold(true)

	m.spans.SetStyles(styles)
}

func (m *Model) spansSetColumns() {
	spans := table.Column{
		Title: "Spans",
		Width: SpanLength + 35,
	}

	functions := table.Column{
		Title: "Functions",
		Width: m.Width - spans.Width + 15,
	}

	m.spans.SetColumns([]table.Column{
		functions,
		spans,
	})
}

func (m *Model) spansSetRows() {
	if m.Current == nil {
		return
	}

	trace := aggregated.Unmarshal(m.Current.Trace)

	sc := span.New(trace.Metadata.ExecutionTime(), float64(SpanLength))

	var rows []table.Row

	for _, s := range trace.Spans {
		rows = append(rows, []string{
			s.Name,
			sc.Render(span.Span{
				Start:    s.Start,
				Duration: s.Elapsed,
			}),
		})
	}

	m.spans.SetRows(rows)
}

func (m *Model) spansView() string {
	return m.spans.View()
}
