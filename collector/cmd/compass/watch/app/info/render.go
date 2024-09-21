package info

import (
	"fmt"

	"github.com/charmbracelet/bubbles/table"

	compasstable "github.com/skpr/compass/collector/cmd/compass/watch/app/table"
)

// View renders the list component.
func (m Model) View() string {
	if len(m.Profiles) == 0 {
		return "No profile data available"
	}

	if len(m.Profiles) < m.Selected {
		return "Incorrect profile selected"
	}

	profile := m.Profiles[m.Selected]

	columns := []table.Column{
		{Title: "Request ID", Width: 45},
		{Title: "Ingestion Time", Width: 35},
		{Title: "Execution Time", Width: 35},
		{Title: "Function Calls", Width: 35},
	}

	var totalInvocations int32

	for _, f := range profile.Functions {
		totalInvocations += f.Invocations
	}

	rows := []table.Row{
		{
			profile.RequestID,
			profile.IngestionTime.Format("15:04:05"),
			fmt.Sprintf("%vms", int(profile.ExecutionTime)),
			fmt.Sprintf("%d", totalInvocations),
		},
	}

	return compasstable.Render(columns, rows, 3)
}
