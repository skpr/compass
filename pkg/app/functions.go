package app

import (
	"fmt"
	"sort"
	"time"

	"github.com/skpr/compass/pkg/app/component/datatable"
	"github.com/skpr/compass/pkg/app/component/span"
	"github.com/skpr/compass/pkg/app/format"
	"github.com/skpr/compass/pkg/app/theme"
	"github.com/skpr/compass/pkg/trace/segmented"
)

// Widths of the functions columns.
const (
	functionsWidthSelf     = 6
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

// SpanSegments is how finely a request is divided when calls are aggregated.
const SpanSegments = 100

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
		{Title: "self", Width: functionsWidthSelf, Align: datatable.AlignRight},
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

func (m *Model) functionsSetRows() {
	if m.Current == nil {
		m.functionSpans = nil
		m.functions.SetRows(nil)

		return
	}

	var (
		executionTime  = m.Current.Metadata.ExecutionTime()
		segmentedTrace = segmented.Unmarshal(m.Current.Trace, SpanSegments)
		timeline       = span.New(time.Duration(executionTime)*time.Nanosecond, functionsWidthTimeline)
	)

	spans := make([]segmented.Span, len(segmentedTrace.Spans))
	copy(spans, segmentedTrace.Spans)

	// Ordered by when each call happened, so the page reads as the request ran.
	// That ordering is most of what a timeline is for: it shows what called
	// what, and where the time went in sequence, rather than presenting a
	// ranking with the causal structure taken out of it.
	//
	// Where two calls start together the longer comes first, which puts a
	// caller above the call it made rather than beneath it.
	sort.SliceStable(spans, func(i, j int) bool {
		if spans[i].StartTime != spans[j].StartTime {
			return spans[i].StartTime < spans[j].StartTime
		}

		if spans[i].Length != spans[j].Length {
			return spans[i].Length > spans[j].Length
		}

		return spans[i].Name < spans[j].Name
	})

	// Kept alongside the rows so the panel below the table can say what the
	// abbreviated name actually was.
	m.functionSpans = spans

	rows := make([]datatable.Row, 0, len(spans))

	for _, s := range spans {
		share := s.SelfShare(executionTime)

		rows = append(rows, datatable.Row{
			functionNameCell(s),
			datatable.Styled(format.Percent(share), theme.S.Severity(theme.ForShare(share))),
			datatable.Styled(format.Bytes(s.MaxMemory), theme.S.CellDim),
			timelineCell(timeline.Bar(span.Span{
				Start:    time.Duration(s.Start) * time.Nanosecond,
				Duration: time.Duration(s.Length) * time.Nanosecond,
				Share:    share,
			})),
			datatable.Styled(format.Duration(s.Length), theme.S.CellDim),
		})
	}

	m.functions.SetRows(rows)
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
func functionNameCell(s segmented.Span) datatable.Cell {
	cell := identifierCell(s.Name)

	if s.TotalFunctionCalls > 1 {
		repeat := fmt.Sprintf(" %s%d", theme.MarkerRepeat, s.TotalFunctionCalls)

		cell.Segments = append(cell.Segments, datatable.Seg(repeat, theme.S.CellFaint))
	}

	return cell
}

func (m *Model) functionsView() string {
	return m.functions.View()
}

// selectedSpan under the cursor, and whether there was one.
func (m *Model) selectedSpan() (segmented.Span, bool) {
	cursor := m.functions.Cursor()

	if cursor < 0 || cursor >= len(m.functionSpans) {
		return segmented.Span{}, false
	}

	return m.functionSpans[cursor], true
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

	self := fmt.Sprintf("%s of %s  %s",
		format.Duration(span.SelfTime),
		format.Duration(executionTime),
		format.Percent(span.SelfShare(executionTime)),
	)

	window := fmt.Sprintf("%s in, ran for %s  %s",
		format.Duration(span.Start),
		format.Duration(span.Length),
		format.Count(span.TotalFunctionCalls, "call", "calls"),
	)

	return []string{
		m.inspectValue("function", span.Name),
		m.inspectValue("self", self),
		m.inspectValue("window", window),
	}
}
