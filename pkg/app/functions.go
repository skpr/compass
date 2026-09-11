package app

import (
	"fmt"
	"sort"
	"time"

	"github.com/skpr/compass/pkg/app/component/datatable"
	"github.com/skpr/compass/pkg/app/component/span"
	"github.com/skpr/compass/pkg/app/format"
	"github.com/skpr/compass/pkg/app/theme"
	"github.com/skpr/compass/pkg/trace"
)

// Widths of the functions columns.
const (
	functionsWidthShare    = 6
	functionsWidthMemory   = 10
	functionsWidthTimeline = 44
	functionsWidthElapsed  = 8
	functionsMinName       = 20
)

// Order in which the functions columns are given up.
const (
	functionsPriorityMemory   = 3
	functionsPriorityTimeline = 2
	functionsPriorityElapsed  = 1
)

func (m *Model) functionsInit() {
	m.functions = datatable.New(
		datatable.WithSelectionRail(),
		datatable.WithEmptyMessage("No function calls were recorded above the extension's threshold."),
	)

	m.functionsSetColumns()
}

func (m *Model) functionsSetColumns() {
	m.functions.SetColumns([]datatable.Column{
		{Title: "function", Flex: 1, MinWidth: functionsMinName},
		{Title: "share", Width: functionsWidthShare, Align: datatable.AlignRight},
		{Title: "mem (inc)", Width: functionsWidthMemory, Align: datatable.AlignRight, Priority: functionsPriorityMemory},
		{Title: m.timelineTitle(), Width: functionsWidthTimeline, Priority: functionsPriorityTimeline},
		{Title: "elapsed", Width: functionsWidthElapsed, Align: datatable.AlignRight, Priority: functionsPriorityElapsed},
	})
}

// timelineTitle is the axis, which doubles as the column's header.
//
// Putting the scale in the header is what turns the timeline from a picture of
// which call came first into something you can read a position off.
func (m *Model) timelineTitle() string {
	return span.New(time.Second, functionsWidthTimeline).Axis()
}

// functionsInvalidateSpans discards the aggregate, so that the next rebuild
// computes it for whichever trace is open then.
func (m *Model) functionsInvalidateSpans() {
	m.functionSpans = nil
	m.functionSpansTrace = nil
	m.functionVisible = nil
}

// functionsEnsureSpans puts the open trace's spans in the order the page shows
// them, once for as long as that trace stays open.
//
// The order is a property of the trace alone: it does not depend on the
// filter, the terminal width or the cursor. Sorting per rebuild meant paying
// for it on every filter keystroke and every resize, for a result which could
// not have changed.
func (m *Model) functionsEnsureSpans() {
	if m.Current == nil {
		m.functionsInvalidateSpans()

		return
	}

	// Opening a trace replaces Current with a new value, so its identity is
	// what says whether the aggregate on hand belongs to it.
	if m.functionSpansTrace == m.Current {
		return
	}

	// Sorted into a copy: the trace belongs to the history, which hands the
	// same value to anything else which asks for it.
	spans := make([]trace.Span, len(m.Current.Spans))
	copy(spans, m.Current.Spans)

	// Ordered by when each call happened, so the page reads as the request ran.
	// That ordering is most of what a timeline is for: it shows what called
	// what, and where the time went in sequence, rather than presenting a
	// ranking with the causal structure taken out of it.
	//
	// Where two calls start together the longer comes first, which puts a
	// caller above the call it made rather than beneath it.
	sort.SliceStable(spans, func(i, j int) bool {
		if spans[i].Offset != spans[j].Offset {
			return spans[i].Offset < spans[j].Offset
		}

		if spans[i].Elapsed != spans[j].Elapsed {
			return spans[i].Elapsed > spans[j].Elapsed
		}

		return spans[i].Name < spans[j].Name
	})

	// Kept alongside the rows so the panel below the table can say what the
	// abbreviated name actually was.
	m.functionSpans = spans
	m.functionSpansTrace = m.Current
}

func (m *Model) functionsSetRows() {
	selectedIndex, preserveSelection := m.selectedFunctionIndex()

	m.functionsEnsureSpans()

	if m.Current == nil {
		m.functions.SetRows(nil)

		return
	}

	var (
		executionTime = m.Current.Metadata.ExecutionTime()
		timeline      = span.New(executionTime, functionsWidthTimeline)
		spans         = m.functionSpans
	)

	values := make([]string, 0, len(spans))
	for _, s := range spans {
		values = append(values, s.Name)
	}
	m.functionVisible = matches(values, m.filterValue(PageFunctions))

	rows := make([]datatable.Row, 0, len(m.functionVisible))

	for _, index := range m.functionVisible {
		s := spans[index]
		share := s.DurationShare(executionTime)

		rows = append(rows, datatable.Row{
			functionNameCell(s),
			datatable.Styled(format.Percent(share), theme.S.Severity(theme.ForShare(share))),
			datatable.Styled(format.Bytes(s.Memory), theme.S.CellDim),
			timelineCell(timeline.Bar(span.Span{
				Start:    s.Offset,
				Duration: s.Elapsed,
			})),
			datatable.Styled(format.Duration(s.Elapsed), theme.S.CellDim),
		})
	}

	m.functions.SetRows(rows)
	if preserveSelection {
		for row, index := range m.functionVisible {
			if index == selectedIndex {
				m.functions.SetCursor(row)
				break
			}
		}
	}
}

// timelineCell of a bar, as segments rather than a rendered string, so that the
// selected row's background survives crossing it.
func timelineCell(bar span.Bar) datatable.Cell {
	return datatable.Join(
		datatable.Seg(bar.Lead, theme.S.Track),
		datatable.Seg(bar.Fill, theme.S.Ramp(bar.Share)),
		datatable.Seg(bar.Trail, theme.S.Track),
	)
}

// functionNameCell, with the repeat count when a span aggregates more than one
// call of the same function.
func functionNameCell(s trace.Span) datatable.Cell {
	cell := identifierCell(s.Name)

	if s.Calls > 1 {
		repeat := fmt.Sprintf(" %s%d", theme.MarkerRepeat, s.Calls)

		cell.Segments = append(cell.Segments, datatable.Seg(repeat, theme.S.CellFaint))
	}

	return cell
}

func (m *Model) functionsView() string {
	return m.functions.View()
}

// selectedSpan under the cursor, and whether there was one.
func (m *Model) selectedFunctionIndex() (int, bool) {
	cursor := m.functions.Cursor()
	if cursor < 0 || cursor >= len(m.functionVisible) {
		return 0, false
	}

	return m.functionVisible[cursor], true
}

func (m *Model) selectedSpan() (trace.Span, bool) {
	index, ok := m.selectedFunctionIndex()
	if !ok || index < 0 || index >= len(m.functionSpans) {
		return trace.Span{}, false
	}

	return m.functionSpans[index], true
}

// functionsInspectLines describe the call under the cursor.
//
// The name in full, because the table abbreviates the namespace to initials and
// then truncates whatever is left — and the namespace is what says which module
// a class came from. Then the numbers behind the two columns which are
// percentages and pictures: what the share is in milliseconds, and where in the
// request the timeline is pointing.
func (m *Model) functionsInspectLines() []string {
	span, ok := m.selectedSpan()
	if !ok {
		return m.inspectMissing("No function calls to inspect.")
	}

	executionTime := m.Current.Metadata.ExecutionTime()

	duration := fmt.Sprintf("%s of %s  %s",
		format.Duration(span.Elapsed),
		format.Duration(executionTime),
		format.Percent(span.DurationShare(executionTime)),
	)

	window := fmt.Sprintf("%s in, ran for %s  %s",
		format.Duration(span.Offset),
		format.Duration(span.Elapsed),
		format.Count(int(span.Calls), "call", "calls"),
	)

	// What the calls in this span cost altogether, which is the number a
	// reader wants when one of them is a function called thousands of times.
	if span.Calls > 1 {
		window = fmt.Sprintf("%s  %s total", window, format.Duration(span.Total))
	}

	return []string{
		m.inspectValue("function", span.Name),
		m.inspectValue("share", duration),
		m.inspectValue("window", window),
	}
}
