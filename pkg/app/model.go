// Package app for handling the main application.
package app

import (
	"github.com/charmbracelet/bubbles/textinput"

	"github.com/skpr/compass/pkg/app/component/datatable"
	"github.com/skpr/compass/pkg/app/events"
	"github.com/skpr/compass/pkg/app/layout"
	"github.com/skpr/compass/pkg/trace"
	"github.com/skpr/compass/pkg/trace/segmented"
)

const (
	// DefaultMaxTraces is how many traces we retain when no limit has been configured.
	DefaultMaxTraces = 500
	// DefaultMaxLogs bounds log history for unattended sessions when no limit
	// has been configured.
	DefaultMaxLogs = 1000
)

// NewModel for executing this application.
func NewModel(probePath string, maxTraces, maxLogs int) *Model {
	if maxTraces <= 0 {
		maxTraces = DefaultMaxTraces
	}
	if maxLogs <= 0 {
		maxLogs = DefaultMaxLogs
	}

	return &Model{
		ProbePath:  probePath,
		MaxTraces:  maxTraces,
		MaxLogs:    maxLogs,
		traces:     newHistory[events.Trace](maxTraces),
		logs:       newHistory[events.Log](maxLogs),
		logEntries: newHistory[logEntry](maxLogs),
	}
}

// Model for storing the state of the application.
type Model struct {
	// Path to the compass.so we are probing.
	ProbePath string

	// MaxTraces is the maximum number of traces we retain, oldest are evicted first.
	MaxTraces int
	// MaxLogs is the maximum number of raw log events we retain, oldest are
	// evicted first. Collapsed rows are derived incrementally from this bound.
	MaxLogs int

	// The current display that is selected.
	PageSelected Page

	// Dimensions.
	Height int
	Width  int

	// Current is the trace which is open, if any.
	Current *events.Trace

	// Collected data is held oldest-to-newest in fixed-capacity rings. Tables
	// present it newest-first without prepending and copying retained events.
	traces     history[events.Trace]
	logs       history[events.Log]
	logEntries history[logEntry]

	// State of the connection to the trace stream.
	connection events.Connection

	// showHelp overlays the key map.
	showHelp bool

	// filter narrows the list on screen. filterFocused is whether it is being
	// typed into; a filter stays in force after the cursor leaves it.
	filter        textinput.Model
	filterFocused bool

	// The rows on the trace pages are cells; these are what the cells were made
	// from, kept so the panel below each table can show a row in full.
	functionSpans []segmented.Span
	drupalEvents  []trace.CacheEvent

	// visible maps a row on screen back to what it came from, so that opening
	// a trace opens the one under the cursor rather than the one at that index
	// in the unfiltered list.
	visible []int

	// regions the screen is divided into, recomputed on resize and on any
	// change which adds or removes a strip.
	regions layout.Regions

	// Tables.
	search    *datatable.Model
	logsTable *datatable.Model
	functions *datatable.Model
	drupal    *datatable.Model
}
