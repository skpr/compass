package app

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"

	"github.com/skpr/compass/tracing/cli/app/color"
)

func (m *Model) metadataInit() {
	m.metadata = table.New(
		table.WithFocused(true),
		table.WithHeight(9),
	)

	m.metadataSetColums()
	m.metadataSetRows()

	styles := table.DefaultStyles()

	styles.Header = styles.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(color.White)).
		BorderBottom(true).
		Bold(false)

	styles.Selected = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(color.White))

	m.metadata.SetStyles(styles)
}

func (m *Model) metadataView() string {
	return m.metadata.View()
}

func (m *Model) metadataSetColums() {
	bold := lipgloss.NewStyle().Bold(true)

	metadata := table.Column{
		Title: bold.Render("Metadata"),
		Width: 20,
	}

	details := table.Column{
		Title: "",
		Width: m.Width - metadata.Width,
	}

	m.metadata.SetColumns([]table.Column{
		metadata,
		details,
	})
}

func (m *Model) metadataSetRows() {
	if m.Current == nil {
		rows := []table.Row{
			{"Trace not selected. Select a trace using the search page."},
		}

		m.metadata.SetRows(rows)

		return
	}

	bold := lipgloss.NewStyle().Bold(true)

	rows := []table.Row{
		{bold.Render("URI"), m.Current.Metadata.URI},
		{bold.Render("Method"), m.Current.Metadata.Method},
		{bold.Render("Execution Time"), fmt.Sprintf("%dms", m.Current.Metadata.ExecutionTime().Milliseconds())},
		{bold.Render("Function Calls"), fmt.Sprintf("%d", len(m.Current.FunctionCalls))},
		{bold.Render("Request ID"), m.Current.Metadata.RequestID},
		{bold.Render("Ingestion Time"), m.Current.IngestionTime.Format(time.RFC822)},
	}

	m.metadata.SetRows(rows)
}
