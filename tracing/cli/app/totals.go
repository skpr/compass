package app

import (
	"fmt"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"

	"github.com/skpr/compass/tracing/cli/app/color"
	"github.com/skpr/compass/tracing/trace/count"
)

func (m *Model) totalsInit() {
	m.totals = table.New(
		table.WithFocused(true),
		table.WithHeight(30),
	)

	m.totalsSetColumns()

	styles := table.DefaultStyles()

	styles.Header = styles.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(color.White)).
		BorderBottom(true).
		Bold(true)

	styles.Selected = styles.Selected.
		Foreground(lipgloss.Color(color.Blue)).
		Bold(true)

	m.totals.SetStyles(styles)
}

func (m *Model) totalsSetColumns() {
	percentage := table.Column{
		Title: "Percentage",
		Width: 24,
	}

	calls := table.Column{
		Title: "Calls",
		Width: 12,
	}

	functions := table.Column{
		Title: "Functions",
		Width: m.Width - calls.Width - percentage.Width,
	}

	m.totals.SetColumns([]table.Column{
		functions,
		calls,
		percentage,
	})
}

func (m *Model) totalsSetRows() {
	if m.Current == nil {
		return
	}

	trace := count.Unmarshal(m.Current.Trace)

	var rows []table.Row

	for _, f := range trace.Functions {
		rows = append(rows, []string{
			f.Name,
			fmt.Sprintf("%d", f.Calls),
			fmt.Sprintf("%d", f.Percentage),
		})
	}

	m.totals.SetRows(rows)
}

func (m *Model) totalsView() string {
	return m.totals.View()
}
