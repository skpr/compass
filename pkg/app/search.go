package app

import (
	"fmt"
	"strings"

	"github.com/skpr/compass/pkg/app/component/datatable"
	"github.com/skpr/compass/pkg/app/events"
	"github.com/skpr/compass/pkg/app/format"
	"github.com/skpr/compass/pkg/app/theme"
	"github.com/skpr/compass/pkg/trace"
	"github.com/skpr/compass/pkg/trace/drupal"
)

// Widths of the search columns.
//
// One line per trace, so that a screen holds twice as many of them as the two
// line rows this replaced. What the second line used to carry — the request
// id, the call count, the age — is either a column now or is on the trace's own
// pages, one keystroke away.
const (
	searchWidthRuntime  = 9
	searchWidthDuration = 8
	searchWidthMemory   = 9
	searchWidthCalls    = 6
	searchWidthID       = 8
	// searchWidthAttention is the marker beside the runtime, one cell wide.
	searchWidthAttention = 1
	searchMinIdentity    = 20
)

// Order in which columns are given up on a narrow terminal. Higher goes first.
//
// The gutter, the duration and what was requested are never dropped: without
// those three a row says nothing at all.
const (
	searchPriorityID     = 3
	searchPriorityCalls  = 2
	searchPriorityMemory = 1
)

func (m *Model) searchInit() {
	m.search = datatable.New(
		datatable.WithSelectionRail(),
		datatable.WithEmptyMessage("Waiting for traces. Generate some traffic and they will appear here."),
		datatable.WithColumns(
			// The marker is never dropped, whatever the terminal is doing: it
			// is the one thing on the row which is not information but a
			// prompt to go and look.
			datatable.Column{Title: "", Width: searchWidthAttention},
			datatable.Column{Title: "runtime", Width: searchWidthRuntime},
			datatable.Column{Title: "duration", Width: searchWidthDuration, Align: datatable.AlignRight},
			datatable.Column{Title: "memory", Width: searchWidthMemory, Align: datatable.AlignRight, Priority: searchPriorityMemory},
			datatable.Column{Title: "calls", Width: searchWidthCalls, Align: datatable.AlignRight, Priority: searchPriorityCalls},
			datatable.Column{Title: "id", Width: searchWidthID, Priority: searchPriorityID},
			datatable.Column{Title: "identity", Flex: 1, MinWidth: searchMinIdentity},
		),
	)
}

// searchSetRows from the retained traces, narrowed by the filter.
func (m *Model) searchSetRows() {
	values := make([]string, 0, m.traces.len())

	for i := 0; i < m.traces.len(); i++ {
		t, _ := m.traces.newest(i)
		values = append(values, t.FilterValue())
	}

	m.visible = matches(values, m.filterValue(PageSearch))

	rows := make([]datatable.Row, 0, len(m.visible))

	for _, index := range m.visible {
		t, _ := m.traces.newest(index)
		rows = append(rows, m.traceRow(t))
	}

	m.search.SetRows(rows)
}

// selectedTrace under the cursor, and whether there was one.
//
// It goes through the visible list rather than indexing the traces directly,
// because a filtered row is not at its own index in the unfiltered slice.
func (m *Model) selectedTrace() (events.Trace, bool) {
	cursor := m.search.Cursor()

	if strings.TrimSpace(m.filterValue(PageSearch)) == "" {
		return m.traces.newest(cursor)
	}

	if cursor < 0 || cursor >= len(m.visible) {
		return events.Trace{}, false
	}

	return m.traces.newest(m.visible[cursor])
}

// traceRow of the search list.
func (m *Model) traceRow(t events.Trace) datatable.Row {
	ms := format.Milliseconds(t.Metadata.ExecutionTime())
	severity := theme.ForDurationMs(ms)

	return datatable.Row{
		attentionCell(t),
		runtimeSourceCell(t.Metadata.Runtime, t.Metadata.Source),
		datatable.Styled(format.Duration(t.Metadata.ExecutionTime()), theme.S.Severity(severity)),
		datatable.Styled(format.Bytes(t.ResourceUtilisation.MaxMemory), theme.S.CellDim),
		datatable.Styled(functionCallCount(t), theme.S.CellDim),
		idCell(t),
		identityCell(t),
	}
}

// runtimeSourceCell names the runtime and how it was invoked, in one column.
//
// Two facts which are always both present and always short, so they cost less
// side by side than they do as two columns with two headers.
func runtimeSourceCell(runtime trace.Runtime, source trace.Source) datatable.Cell {
	return datatable.Join(
		runtimeCell(runtime).Segments[0],
		datatable.Seg(":", theme.S.CellFaint),
		sourceCell(source).Segments[0],
	)
}

// runtimeCell naming which runtime produced the trace.
func runtimeCell(runtime trace.Runtime) datatable.Cell {
	switch runtime {
	case trace.RuntimePHP:
		return datatable.Styled(string(runtime), theme.S.RuntimePHP)
	case trace.RuntimeNode:
		return datatable.Styled(string(runtime), theme.S.RuntimeNode)
	default:
		return datatable.Styled(string(runtime), theme.S.CellFaint)
	}
}

// sourceCell naming how the trace was invoked.
//
// HTTP is grey rather than coloured. It is almost every row, so giving it a hue
// spends the reader's attention on the case which tells them nothing; only the
// unusual one is worth a colour.
func sourceCell(source trace.Source) datatable.Cell {
	switch source {
	case trace.SourceHTTP:
		return datatable.Styled(string(source), theme.S.SourceHTTP)
	case trace.SourceCLI:
		return datatable.Styled(string(source), theme.S.SourceCLI)
	default:
		return datatable.Styled(string(source), theme.S.CellFaint)
	}
}

// shortID of a request, which is thirty-two characters of hex nobody reads in
// full but which has to be recognisable and greppable.
func shortID(id string) string {
	const short = 8

	if len(id) <= short {
		return id
	}

	return id[:short]
}

// idCell of a trace.
//
// The request id is what ties a trace to everything else which logged during
// it, which is the whole reason to carry it on the list rather than only inside
// the trace. Shortened to its first characters: the full value is thirty-two
// characters of hex which nobody reads, but eight is enough to recognise one
// and enough to grep a log with.
//
// A trace which arrived without an X-Request-ID header has no id at all, and a
// column reading UNKNOWN on every row would be worse than an empty one.
func idCell(t events.Trace) datatable.Cell {
	if !t.Metadata.Identified() {
		return datatable.Styled(theme.MarkerSeparator, theme.S.CellFaint)
	}

	return datatable.Styled(shortID(t.Metadata.ID), theme.S.CellDim)
}

// Markers for partial function-call data.
const (
	// TruncatedMarker after a call count means additional calls were dropped.
	TruncatedMarker = "+"
	// PartialTimingMarker means derived timing uses retained calls only.
	PartialTimingMarker = "*"
)

// functionCallCount reports retained calls and marks partial traces.
func functionCallCount(t events.Trace) string {
	marker := ""
	if t.FunctionCallsDropped > 0 {
		marker = TruncatedMarker
	}

	return fmt.Sprintf("%d%s", len(t.FunctionCalls), marker)
}

// AttentionMarker is set beside the runtime on a request worth going and
// looking at.
//
// A mark rather than a letter. Two letter codes in a column of their own were
// a puzzle to anyone who had not read the legend — you have to know what they
// stand for before they tell you anything, which is the wrong way round for
// something whose whole job is to catch the eye. A red dot needs no key.
const AttentionMarker = theme.MarkerPresent

// attentionCell for a trace, set when the response cannot be cached.
//
// Only that one condition. A trace whose events were dropped is truncated,
// which is worth knowing once it is open — the strip above the trace says so —
// but it is not a reason to open it, and a marker which fires for two different
// reasons stops meaning either of them.
func attentionCell(t events.Trace) datatable.Cell {
	if !drupal.IsUncacheable(t.Trace) {
		return datatable.Text(" ")
	}

	return datatable.Styled(AttentionMarker, theme.S.Severity(theme.LevelCritical))
}

// identityCell of a trace: what was actually requested.
func identityCell(t events.Trace) datatable.Cell {
	switch t.Metadata.Source {
	case trace.SourceHTTP:
		// The method is a label on the path, not a value of its own, so it sits
		// a rung below it.
		return datatable.Join(
			datatable.Seg(t.Metadata.HTTP.Method+" ", theme.S.CellFaint),
			datatable.Seg(t.Metadata.HTTP.URI, theme.S.Cell),
		)
	case trace.SourceCLI:
		return datatable.Styled(t.Metadata.CLI.Command, theme.S.Cell)
	default:
		return datatable.Styled(t.Metadata.ID, theme.S.CellFaint)
	}
}

func (m *Model) searchView() string {
	return m.search.View()
}
