package app

import (
	"fmt"
	"strings"
	"time"

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

// logsSetRows from the collected log events, collapsing repeats.
func (m *Model) logsSetRows() {
	entries := collapseLogs(m.logs)

	values := make([]string, 0, len(entries))
	for _, entry := range entries {
		values = append(values, entry.Message)
	}

	rows := make([]datatable.Row, 0, len(entries))

	for _, index := range matches(values, m.filter.Value()) {
		entry := entries[index]

		severity := logSeverity(entry.Type)

		rows = append(rows, datatable.Row{
			datatable.Styled(entry.Time.Local().Format(logTimeFormat), theme.S.CellDim),
			datatable.Styled(strings.ToUpper(entry.Type), theme.S.Severity(severity)),
			repeatCell(entry.Repeats),
			datatable.Styled(entry.Message, theme.S.Cell),
		})
	}

	m.logsTable.SetRows(rows)
}

// logEntry is a log event and how many times it repeated.
type logEntry struct {
	events.Log

	// Repeats counts this occurrence and every identical one after it.
	Repeats int
}

// collapseLogs folds runs of the same message into one row with a count.
//
// A sidecar which cannot be reached logs the same line every retry, so a screen
// of logs is otherwise forty copies of one sentence and none of the others.
// The count says how bad it is; one row says what it is.
func collapseLogs(logs []events.Log) []logEntry {
	entries := make([]logEntry, 0, len(logs))

	for _, log := range logs {
		if len(entries) > 0 {
			previous := &entries[len(entries)-1]

			if previous.Message == log.Message && previous.Type == log.Type {
				previous.Repeats++

				// The most recent occurrence is the one worth timestamping.
				if log.Time.After(previous.Time) {
					previous.Time = log.Time
				}

				continue
			}
		}

		entries = append(entries, logEntry{Log: log, Repeats: 1})
	}

	return entries
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

// logsSince reports how long ago the most recent log arrived.
func (m *Model) logsSince(now time.Time) (time.Duration, bool) {
	if len(m.logs) == 0 {
		return 0, false
	}

	return now.Sub(m.logs[0].Time), true
}

func (m *Model) logsView() string {
	return m.logsTable.View()
}
