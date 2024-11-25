package info

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"

	compasstable "github.com/skpr/compass/cli/app/components/table"
)

// View renders the list component.
func (m Model) View() string {
	if len(m.Traces) == 0 {
		return "No profile data available"
	}

	if len(m.Traces) < m.Selected {
		return "Incorrect profile selected"
	}

	profile := m.Traces[m.Selected]

	trace := compasstable.Render([]table.Column{
		{Title: "Request ID", Width: 45},
		{Title: "Ingested Time", Width: 35},
		{Title: "Execution Time", Width: 35},
		{Title: "Function Calls", Width: 35},
	}, []table.Row{
		{
			profile.RequestID,
			time.UnixMicro(profile.StartTime).Format(time.TimeOnly),
			fmt.Sprintf("%vms", int(profile.ExecutionTime)),
			fmt.Sprintf("%d", len(profile.FunctionCalls)),
		},
	}, 4)

	request := compasstable.Render([]table.Column{
		{Title: "Method", Width: 45},
		{Title: "URI", Width: 109},
	}, []table.Row{
		{
			profile.Method,
			profile.URI,
		},
	}, 3)

	return lipgloss.JoinVertical(lipgloss.Top, trace, request)
}
