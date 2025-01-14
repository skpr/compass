package count

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/bubbles/table"

	compasstable "github.com/skpr/compass/cli/app/components/table"
	"github.com/skpr/compass/trace/count"
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

	trace := count.Unmarshal(m.Trace)

	var visible []table.Row

	start, end := compasstable.GetPositionStartAndEnd(m.ScrollPosition, VisibleRows, len(trace.Functions))

	for i, s := range trace.Functions {
		if i < start || i >= end {
			continue
		}

		visible = append(visible, []string{
			s.Name,
			strconv.Itoa(s.Calls),
			fmt.Sprintf("%s%%", strconv.Itoa(s.Percentage)),
		})
	}

	columns := []table.Column{
		{Title: "Function", Width: 106},
		{Title: "Calls", Width: 26},
		{Title: "Percentage", Width: 24},
	}

	// We add 2 to account for the header and the border.
	return compasstable.Render(columns, visible, VisibleRows+2)
}
