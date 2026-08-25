package app

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"

	"github.com/skpr/compass/pkg/app/component/datatable"
	"github.com/skpr/compass/pkg/app/component/text"
	"github.com/skpr/compass/pkg/app/format"
	"github.com/skpr/compass/pkg/app/theme"
	"github.com/skpr/compass/pkg/trace"
	"github.com/skpr/compass/pkg/trace/drupal"
)

// Widths of the Drupal columns.
//
// Max age is ten cells for a value which is at most nine. It used to be
// twenty-four, because the old table measured a cell with its escape sequences
// included and a coloured value had to fit the column with them; that whole
// class of workaround went with the table.
const (
	drupalWidthMaxAge   = 10
	drupalWidthCalls    = 6
	drupalWidthObject   = 20
	drupalWidthTags     = 6
	drupalWidthContexts = 6
	drupalWidthOrigin   = 11
	drupalMinCaller     = 20
)

// Order in which the Drupal columns are given up.
const (
	drupalPriorityOrigin   = 4
	drupalPriorityObject   = 3
	drupalPriorityContexts = 2
	drupalPriorityTags     = 1
)

func (m *Model) drupalInit() {
	m.drupal = datatable.New(
		datatable.WithSelectionRail(),
		datatable.WithEmptyMessage("No Drupal cacheability was collected for this trace."),
		datatable.WithColumns(
			datatable.Column{Title: "caller", Flex: 1, MinWidth: drupalMinCaller},
			datatable.Column{Title: "max age", Width: drupalWidthMaxAge, Align: datatable.AlignRight},
			datatable.Column{Title: "calls", Width: drupalWidthCalls, Align: datatable.AlignRight},
			datatable.Column{Title: "object", Width: drupalWidthObject, Priority: drupalPriorityObject},
			datatable.Column{Title: "tags", Width: drupalWidthTags, Align: datatable.AlignRight, Priority: drupalPriorityTags},
			datatable.Column{Title: "ctx", Width: drupalWidthContexts, Align: datatable.AlignRight, Priority: drupalPriorityContexts},
			datatable.Column{Title: "origin", Width: drupalWidthOrigin, Priority: drupalPriorityOrigin},
		),
	)
}

func (m *Model) drupalSetRows() {
	summary := m.drupalSummary()

	if !summary.Collected {
		m.drupalEvents = nil
		m.drupal.SetRows(nil)

		return
	}

	// Kept alongside the rows so the panel below the table can show the values
	// which are too long to sit in a column. A row is cells; this is what the
	// cells were made from.
	m.drupalEvents = summary.Events

	rows := make([]datatable.Row, 0, len(summary.Events))

	for _, event := range summary.Events {
		rows = append(rows, datatable.Row{
			identifierCell(event.Caller),
			datatable.Styled(format.MaxAge(event.MaxAge), maxAgeStyle(event.MaxAge)),
			datatable.Styled(fmt.Sprintf("%d", event.Calls), theme.S.CellDim),
			datatable.Styled(shortenObjectType(event.ObjectType), theme.S.Cell),
			countCell(len(event.Tags)),
			countCell(len(event.Contexts)),
			datatable.Styled(string(event.Origin), theme.S.CellFaint),
		})
	}

	m.drupal.SetRows(rows)
}

// maxAgeStyle for a cacheability value.
//
// Zero is what made the response uncacheable, so it is the loudest thing on the
// row; permanent is the good case and recedes.
func maxAgeStyle(maxAge int64) lipgloss.Style {
	switch {
	case maxAge == 0:
		return theme.S.Severity(theme.LevelCritical).Bold(true)
	case maxAge == trace.CacheMaxAgePermanent:
		return theme.S.CellFaint
	default:
		return theme.S.Severity(theme.ForMaxAge(maxAge))
	}
}

// countCell of a tag or context list.
//
// The lists themselves used to be joined into the column and truncated, which
// produced an unreadable fragment and stole forty six columns from the caller
// name — the thing the reader actually came for. The count says how much there
// is; the trace itself says what.
func countCell(count int) datatable.Cell {
	if count == 0 {
		return datatable.Styled("0", theme.S.CellFaint)
	}

	return datatable.Styled(fmt.Sprintf("%d", count), theme.S.CellDim)
}

// shortenObjectType to its class name. The namespace is the same for most of
// what Drupal reports here, so it costs a lot of column width to say little —
// and the panel below the table carries the whole name.
func shortenObjectType(objectType string) string {
	_, class, _ := text.Identifier(objectType)

	return class
}

// drupalSummary of the selected trace, or the zero value when there is none.
func (m *Model) drupalSummary() drupal.Summary {
	if m.Current == nil {
		return drupal.Summary{}
	}

	return drupal.Unmarshal(m.Current.Trace)
}

func (m *Model) drupalView() string {
	return m.drupal.View()
}

// selectedCacheEvent under the cursor, and whether there was one.
func (m *Model) selectedCacheEvent() (trace.CacheEvent, bool) {
	cursor := m.drupal.Cursor()

	if cursor < 0 || cursor >= len(m.drupalEvents) {
		return trace.CacheEvent{}, false
	}

	return m.drupalEvents[cursor], true
}

// drupalInspectLines describe the cacheability event under the cursor.
func (m *Model) drupalInspectLines() []string {
	event, ok := m.selectedCacheEvent()
	if !ok {
		return m.inspectMissing("No cacheability to inspect.")
	}

	return []string{
		m.inspectValue("caller", event.Caller),
		m.inspectValue("object", event.ObjectType),
		m.inspectList("tags", event.Tags),
		m.inspectList("contexts", event.Contexts),
	}
}
