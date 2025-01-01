package spans

import (
	"github.com/charmbracelet/bubbles/table"
	"github.com/skpr/compass/trace/segmented"

	compasstable "github.com/skpr/compass/cli/app/components/table"
	"github.com/skpr/compass/cli/app/components/view/spans/graph"
)

const (
	// VisibleRows is the number of rows that are visible in the breakdown.
	VisibleRows = 30
)

// View refreshes the display for the breakdown.
func (m Model) View() string {
	if len(m.Trace.FunctionCalls) == 0 {
		return "No tracing data available"
	}

	trace := segmented.Unmarshal(m.Trace, 50) // @todo, Should not be hardcoded.

	var visible []table.Row

	start, end := getPositionStartAndEnd(m.ScrollPosition, VisibleRows, len(trace.Spans))

	for i, s := range trace.Spans {
		if i < start || i >= end {
			continue
		}

		visible = append(visible, []string{
			s.GetName(),
			graph.Render(s.Start, s.Length),
		})
	}

	columns := []table.Column{
		{Title: "Function", Width: 106},
		{Title: "Span", Width: 52},
	}

	// We add 2 to account for the header and the border.
	return compasstable.Render(columns, visible, VisibleRows+2)
}
