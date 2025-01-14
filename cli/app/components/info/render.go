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
	if len(m.Trace.FunctionCalls) == 0 {
		return "No profile data available"
	}

	trace := compasstable.Render([]table.Column{
		{Title: "Request ID", Width: 45},
		{Title: "Ingested Time", Width: 35},
		{Title: "Execution Time", Width: 35},
		{Title: "Function Calls", Width: 35},
	}, []table.Row{
		{
			m.Trace.Metadata.RequestID,
			time.UnixMicro(m.Trace.Metadata.StartTime).Format(time.TimeOnly),
			fmt.Sprintf("%vms", int(m.Trace.Metadata.ExecutionTime)),
			fmt.Sprintf("%d", len(m.Trace.FunctionCalls)),
		},
	}, 4)

	request := compasstable.Render([]table.Column{
		{Title: "Method", Width: 45},
		{Title: "URI", Width: 109},
	}, []table.Row{
		{
			m.Trace.Metadata.Method,
			m.Trace.Metadata.URI,
		},
	}, 3)

	return lipgloss.JoinVertical(lipgloss.Top, trace, request)
}
