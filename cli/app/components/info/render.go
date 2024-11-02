package info

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/table"

	compasstable "github.com/skpr/compass/cli/app/components/table"
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
		{Title: "Ingested Time", Width: 35},
		{Title: "Execution Time", Width: 35},
		{Title: "Function Calls", Width: 35},
	}

	rows := []table.Row{
		{
			profile.RequestID,
			time.UnixMicro(profile.StartTime).Format(time.TimeOnly),
			fmt.Sprintf("%vms", int(profile.ExecutionTime)),
			fmt.Sprintf("%d", len(profile.FunctionCalls)),
		},
	}

	return compasstable.Render(columns, rows, 3)
}
