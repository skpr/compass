package app

import (
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"

	"github.com/skpr/compass/cli/app/color"
	"github.com/skpr/compass/cli/app/component/span"
	"github.com/skpr/compass/trace/segmented"
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
		Width: 90,
	}

	functions := table.Column{
		Title: "Functions",
		Width: 100,
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

	//length := int64(SpanLength - 2)

	trace := segmented.Unmarshal(m.Current.Trace, SpanLength)

	sc := span.New(time.Duration(trace.Metadata.ExecutionTime())*time.Nanosecond, float64(SpanLength))

	var rows []table.Row

	for _, s := range trace.Spans {
		rows = append(rows, []string{
			s.Name,
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
