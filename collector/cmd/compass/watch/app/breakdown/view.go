package breakdown

import (
	"fmt"
	"sort"

	"github.com/charmbracelet/bubbles/table"

	compasstable "github.com/skpr/compass/collector/cmd/compass/watch/app/table"
)

const (
	// VisibleRows is the number of rows that are visible in the breakdown.
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

	var rows []Row

	for name, details := range profile.Functions {
		rows = append(rows, Row{
			Name:          name,
			ExecutionTime: details.ExecutionTime,
			Invocations:   details.Invocations,
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		return rows[i].ExecutionTime > rows[j].ExecutionTime
	})

	var visible []table.Row

	start, end := getPositionStartAndEnd(m.ScrollPosition, VisibleRows, len(profile.Functions))

	for i, f := range rows {
		if i < start || i >= end {
			continue
		}

		visible = append(visible, []string{
			f.Name,
			getExecutionGraph(profile.ExecutionTime, f.ExecutionTime),
			fmt.Sprintf("%d", f.Invocations),
		})
	}

	columns := []table.Column{
		{Title: "Function", Width: 109},
		{Title: "Execution Time", Width: 30},
		{Title: "Invocations", Width: 15},
	}

	// We add 2 to account for the header and the border.
	return compasstable.Render(columns, visible, VisibleRows+2)
}
