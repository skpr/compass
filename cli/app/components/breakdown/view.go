package breakdown

import (
	"sort"

	"github.com/charmbracelet/bubbles/table"

	"github.com/skpr/compass/cli/app/components/breakdown/graph"
	compasstable "github.com/skpr/compass/cli/app/components/table"
)

const (
	// VisibleRows is the number of rows that are visible in the breakdown.
	VisibleRows = 30
)

// View refreshes the display for the breakdown.
func (m Model) View() string {
	if len(m.Traces) == 0 {
		return "No profile data available"
	}

	if len(m.Traces) < m.Selected {
		return "Incorrect profile selected"
	}

	profile := m.Traces[m.Selected]

	if len(profile.FunctionCalls) == 0 {
		return "No functions available for profile"
	}

	var rows []Row

	for _, details := range profile.FunctionCalls {
		rows = append(rows, Row{
			Name:      details.Name,
			StartTime: details.StartTime,
			EndTime:   details.EndTime,
			Diff:      details.EndTime - details.StartTime,
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		return rows[i].StartTime < rows[j].StartTime
	})

	var visible []table.Row

	start, end := getPositionStartAndEnd(m.ScrollPosition, VisibleRows, len(profile.FunctionCalls))

	for i, f := range rows {
		if i < start || i >= end {
			continue
		}

		visible = append(visible, []string{
			f.Name,
			graph.Render(profile.StartTime, f.StartTime, profile.ExecutionTime, f.Diff/1000),
		})
	}

	columns := []table.Column{
		{Title: "Function", Width: 106},
		{Title: "Span", Width: 52},
	}

	// We add 2 to account for the header and the border.
	return compasstable.Render(columns, visible, VisibleRows+2)
}
