package breakdown

import (
	"fmt"
	"sort"

	"github.com/charmbracelet/bubbles/table"

	compasstable "github.com/skpr/compass/collector/cmd/compass/watch/app/table"
)

const (
	VisibleRows = 30
)

// View refreshes the display for the breakdown.
func (m Model) View() string {
	if len(m.Profiles) == 0 {
		return "No profile data available"
	}

	if len(m.Profiles) < m.Selected {
		return "Incorrect profile selected"
	}

	profile := m.Profiles[m.Selected]

	if len(profile.Functions) == 0 {
		return "No functions available for profile"
	}

	sort.Slice(profile.Functions, func(i, j int) bool {
		return profile.Functions[i].TotalExecutionTime > profile.Functions[j].TotalExecutionTime
	})

	var rows []table.Row

	start, end := getPositionStartAndEnd(m.ScrollPosition, VisibleRows, len(profile.Functions))

	for i, f := range profile.Functions {
		if i < start || i >= end {
			continue
		}

		rows = append(rows, []string{
			f.Name,
			getExecutionGraph(profile.ExecutionTime, f.TotalExecutionTime),
			fmt.Sprintf("%d", f.Invocations),
		})
	}

	columns := []table.Column{
		{Title: "Function", Width: 109},
		{Title: "Execution Time", Width: 30},
		{Title: "Invocations", Width: 15},
	}

	// We add 2 to account for the header and the border.
	return compasstable.Render(columns, rows, VisibleRows+2)
}
