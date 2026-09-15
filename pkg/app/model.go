// Package app for handling the main application.
package app

import (
	"github.com/charmbracelet/bubbles/textinput"

	"github.com/skpr/compass/pkg/app/component/datatable"
	"github.com/skpr/compass/pkg/app/events"
	"github.com/skpr/compass/pkg/app/layout"
	"github.com/skpr/compass/pkg/trace"
)

const (
	// DefaultMaxTraces is how many traces we retain when no limit has been configured.
	DefaultMaxTraces = 500
	// DefaultMaxLogs bounds log history for unattended sessions when no limit
	// has been configured.
	DefaultMaxLogs = 1000
	// DefaultMaxBytes bounds what the retained traces may weigh.
	//
	// A count on its own does not bound memory: traces differ by orders of
	// magnitude, from a handful of spans to the thousands a request making a
	// million calls produces, so the same five hundred traces are anywhere
	// between a few megabytes and most of a gigabyte.
	DefaultMaxBytes = 256 << 20
)

// Options for the application.
type Options struct {
	// MaxTraces retained, oldest discarded first.
	MaxTraces int
	// MaxLogs retained, oldest discarded first.
	MaxLogs int
	// MaxBytes the retained traces may weigh, oldest discarded first. The
	// newest trace is always kept, however large it is.
	MaxBytes int
}

// NewModel for executing this application. Non-positive options use the
// defaults above.
func NewModel(probePath string, options Options) *Model {
	if options.MaxTraces <= 0 {
		options.MaxTraces = DefaultMaxTraces
	}
	if options.MaxLogs <= 0 {
		options.MaxLogs = DefaultMaxLogs
	}
	if options.MaxBytes <= 0 {
		options.MaxBytes = DefaultMaxBytes
	}

	return &Model{
		ProbePath:  probePath,
		MaxTraces:  options.MaxTraces,
		MaxLogs:    options.MaxLogs,
		MaxBytes:   options.MaxBytes,
		traces:     newHistory[events.Trace](options.MaxTraces),
		logs:       newHistory[events.Log](options.MaxLogs),
		logEntries: newHistory[logEntry](options.MaxLogs),
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
	// MaxBytes is what the retained traces may weigh before the oldest are
	// evicted, whatever MaxTraces allows.
	MaxBytes int

	// The current display that is selected.
	PageSelected Page

	// Dimensions.
	Height int
	Width  int

	// Current is the trace which is open, if any.
	Current *events.Trace

	// Collected data is held oldest-to-newest in fixed-capacity rings. Tables
	// present it newest-first without prepending and copying retained events.
	traces history[events.Trace]
	// tracesBytes is what the retained traces weigh, kept as they arrive and
	// leave rather than walked for.
	tracesBytes int
	logs        history[events.Log]
	logEntries  history[logEntry]

	// State of the connection to the trace stream.
	connection events.Connection

	// showHelp overlays the key map.
	showHelp bool

	// A trace-closing Escape is briefly deferred because terminal navigation
	// keys are ESC-prefixed sequences which can be split across input reads.
	traceClosePending  bool
	traceCloseSequence uint64

	// filter narrows the list on screen. filterFocused is whether it is being
	// typed into; a filter stays in force after the cursor leaves it. Search
	// and Logs share a value, while each trace page keeps its own so a URI
	// query does not accidentally hide every function in an opened trace.
	filter               textinput.Model
	filterFocused        bool
	listFilterValue      string
	functionsFilterValue string
	drupalFilterValue    string

	// The rows on the trace pages are cells; these are what the cells were made
	// from, and the visible maps take a filtered table row back to that source
	// so the panel below it still describes the selected row.
	functionSpans   []trace.Span
	functionVisible []int
	// functionSpansTrace is the trace functionSpans was aggregated from, so a
	// rebuild can tell an aggregate it can reuse from one belonging to a trace
	// which is no longer open.
	functionSpansTrace *events.Trace
	drupalEvents       []trace.CacheEvent
	drupalVisible      []int

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
