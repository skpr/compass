package app

import (
	"fmt"

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
		Foreground(lipgloss.Color(color.Blue)).
		Bold(true)

	m.spans.SetStyles(styles)
}

func (m *Model) spansSetColumns() {
	calls := table.Column{
		Title: "Calls",
		Width: 12,
	}

	spans := table.Column{
		Title: "Spans",
		Width: SpanLength,
	}

	functions := table.Column{
		Title: "Functions",
		Width: m.Width - calls.Width - spans.Width,
	}

	m.spans.SetColumns([]table.Column{
		functions,
		spans,
		calls,
	})
}

func (m *Model) spansSetRows() {
	if m.Current == nil {
		return
	}

	length := int64(SpanLength - 2)

	trace := segmented.Unmarshal(m.Current.Trace, length)

	var rows []table.Row

	for _, s := range trace.Spans {
		rows = append(rows, []string{
			s.Name,
			span.Render(s.Start, s.Length, length),
			fmt.Sprintf("%d", s.TotalFunctionCalls),
		})
	}

	m.spans.SetRows(rows)
}

func (m *Model) spansView() string {
	return m.spans.View()
}
