package app

import (
	"fmt"
	"strings"

	"github.com/skpr/compass/pkg/app/component/datatable"
	"github.com/skpr/compass/pkg/app/events"
	"github.com/skpr/compass/pkg/app/theme"
)

// Widths of the log columns.
const (
	logsWidthTime   = 12
	logsWidthLevel  = 5
	logsWidthRepeat = 5
	logsMinMessage  = 20
)

// Order in which the log columns are given up.
const (
	logsPriorityRepeat = 2
	logsPriorityLevel  = 1
)

// logTimeFormat is the time of day with milliseconds.
//
// The full date on every row was twenty-nine characters of which twenty-six
// were identical from one line to the next. What changes between two log lines
// is the time of day, so that is what the column shows.
const logTimeFormat = "15:04:05.000"

func (m *Model) logsInit() {
	m.logsTable = datatable.New(
		datatable.WithSelectionRail(),
		datatable.WithEmptyMessage("Nothing to report."),
		datatable.WithColumns(
			datatable.Column{Title: "time", Width: logsWidthTime},
			datatable.Column{Title: "level", Width: logsWidthLevel, Priority: logsPriorityLevel},
			datatable.Column{Title: "", Width: logsWidthRepeat, Align: datatable.AlignRight, Priority: logsPriorityRepeat},
			datatable.Column{Title: "message", Flex: 1, MinWidth: logsMinMessage},
		),
	)
}

// logsSetRows rebuilds visible rows from the incrementally collapsed history.
func (m *Model) logsSetRows() {
	entries := make([]logEntry, 0, m.logEntries.len())
	values := make([]string, 0, m.logEntries.len())

	for i := 0; i < m.logEntries.len(); i++ {
		entry, _ := m.logEntries.newest(i)
		entries = append(entries, entry)
		values = append(values, entry.Message)
	}

	rows := make([]datatable.Row, 0, len(entries))
	for _, index := range matches(values, m.filterValue(PageLogs)) {
		rows = append(rows, m.logRow(entries[index]))
	}

	m.logsTable.SetRows(rows)
}

func (m *Model) logRow(entry logEntry) datatable.Row {
	severity := logSeverity(entry.Type)

	return datatable.Row{
		datatable.Styled(entry.Time.Local().Format(logTimeFormat), theme.S.CellDim),
		datatable.Styled(strings.ToUpper(entry.Type), theme.S.Severity(severity)),
		repeatCell(entry.Repeats),
		datatable.Styled(entry.Message, theme.S.Cell),
	}
}

// logEntry is a log event and how many times it repeated.
type logEntry struct {
	events.Log

	// Repeats counts this occurrence and every identical one after it.
	Repeats int
}

// repeatCell of a collapsed run.
func repeatCell(repeats int) datatable.Cell {
	if repeats <= 1 {
		return datatable.Text("")
	}

	return datatable.Styled(fmt.Sprintf("%s%d", theme.MarkerRepeat, repeats), theme.S.CellFaint)
}

// logSeverity of a log level.
func logSeverity(level string) theme.Severity {
	switch strings.ToLower(level) {
	case "error":
		return theme.LevelCritical
	case "warn", "warning":
		return theme.LevelWarn
	default:
		return theme.LevelNone
	}
}

func (m *Model) logsView() string {
	return m.logsTable.View()
}
