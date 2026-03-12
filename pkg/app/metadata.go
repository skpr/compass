package app

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"

	"github.com/skpr/compass/pkg/app/color"
	skprtime "github.com/skpr/compass/pkg/app/time"
	"github.com/skpr/compass/pkg/trace"
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

	var rows []table.Row

	switch m.Current.Metadata.Source {
	case trace.SourceHTTP:
		rows = []table.Row{
			{bold.Render("URI"), m.Current.Metadata.HTTP.URI},
			{bold.Render("Method"), m.Current.Metadata.HTTP.Method},
			{bold.Render("Execution Time"), fmt.Sprintf("%dms", skprtime.NanosecondsToMilliseconds(m.Current.Metadata.ExecutionTime()))},
			{bold.Render("Function Calls"), fmt.Sprintf("%d", len(m.Current.FunctionCalls))},
			{bold.Render("Max Memory"), formatBytes(m.Current.ResourceUtilisation.MaxMemory)},
			{bold.Render("Request ID"), m.Current.Metadata.ID},
			{bold.Render("Ingestion Time"), m.Current.IngestionTime.Format(time.RFC822)},
		}
	case trace.SourceCLI:
		rows = []table.Row{
			{bold.Render("Command"), m.Current.Metadata.CLI.Command},
			{bold.Render("Execution Time"), fmt.Sprintf("%dms", skprtime.NanosecondsToMilliseconds(m.Current.Metadata.ExecutionTime()))},
			{bold.Render("Function Calls"), fmt.Sprintf("%d", len(m.Current.FunctionCalls))},
			{bold.Render("Max Memory"), formatBytes(m.Current.ResourceUtilisation.MaxMemory)},
			{bold.Render("Process ID"), m.Current.Metadata.ID},
			{bold.Render("Ingestion Time"), m.Current.IngestionTime.Format(time.RFC822)},
		}
	default:
		rows = []table.Row{
			{bold.Render("Source"), string(m.Current.Metadata.Source)},
			{bold.Render("Execution Time"), fmt.Sprintf("%dms", skprtime.NanosecondsToMilliseconds(m.Current.Metadata.ExecutionTime()))},
			{bold.Render("Function Calls"), fmt.Sprintf("%d", len(m.Current.FunctionCalls))},
			{bold.Render("Max Memory"), formatBytes(m.Current.ResourceUtilisation.MaxMemory)},
			{bold.Render("ID"), m.Current.Metadata.ID},
			{bold.Render("Ingestion Time"), m.Current.IngestionTime.Format(time.RFC822)},
		}
	}

	m.metadata.SetRows(rows)
}
