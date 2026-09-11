package app

import (
	"fmt"
	"testing"
	"time"
	"unsafe"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skpr/compass/pkg/app/events"
	"github.com/skpr/compass/pkg/trace"
)

// functionsTrace of a request carrying calls, spread over a one second request
// so that they land in different segments.
func functionsTrace(id string, calls int) events.Trace {
	start := time.Unix(1700000000, 0)

	fullTrace := trace.Trace{
		Metadata: trace.Metadata{
			ID:        id,
			Source:    trace.SourceHTTP,
			Runtime:   trace.RuntimePHP,
			HTTP:      trace.MetadataHTTP{Method: "GET", URI: "/" + id},
			StartTime: start,
			EndTime:   start.Add(time.Second),
		},
		FunctionCalls: make([]trace.FunctionCall, 0, calls),
	}

	for i := range calls {
		fullTrace.FunctionCalls = append(fullTrace.FunctionCalls, trace.FunctionCall{
			Name:    fmt.Sprintf("Drupal\\Core\\Entity\\Sql\\SqlContentEntityStorage%d::loadMultiple", i%400),
			Offset:  time.Duration(i) * time.Microsecond % time.Second,
			Elapsed: time.Duration(i%500) * time.Microsecond,
			Memory:  int64(i) * 128,
		})
	}

	return events.Trace{Trace: fullTrace}
}

// functionsModel with one trace open on the Functions page.
func functionsModel(t *testing.T, calls int) *Model {
	t.Helper()

	m := NewModel("/tmp/compass.so", 10, 10)
	m.Init()
	m.Height, m.Width = 40, 160

	m.updateTrace(functionsTrace("first", calls))
	m.updateKeyEnter()

	require.Equal(t, PageFunctions, m.PageSelected)
	require.NotEmpty(t, m.functionSpans)

	return m
}

// The aggregate depends on the trace alone, so rebuilding the rows for a
// different filter must reuse it rather than segment the calls again.
func TestFunctionsSetRows_ReusesTheAggregateWhileTheTraceIsOpen(t *testing.T) {
	m := functionsModel(t, 2_000)

	first := unsafe.SliceData(m.functionSpans)
	spans := len(m.functionSpans)

	m.storeFilter(PageFunctions, "loadMultiple")
	m.functionsSetRows()

	assert.Same(t, first, unsafe.SliceData(m.functionSpans), "filtering re-segmented the trace")
	assert.Len(t, m.functionSpans, spans, "filtering changed the aggregate")
	assert.NotEmpty(t, m.functionVisible, "the filter matched nothing, so it proves nothing")

	// A resize rebuilds the rows, which carry their own widths, but not the
	// aggregate they are built from.
	m.functionsSetRows()

	assert.Same(t, first, unsafe.SliceData(m.functionSpans), "a rebuild re-segmented the trace")
}

// Opening another trace has to replace the aggregate, or the page would show
// the previous request's calls.
func TestFunctionsSetRows_RecomputesForAnotherTrace(t *testing.T) {
	m := functionsModel(t, 2_000)

	first := unsafe.SliceData(m.functionSpans)

	m.updateKeyEsc()
	m.updateTrace(functionsTrace("second", 10))
	m.search.SetCursor(0)
	m.updateKeyEnter()

	require.Equal(t, "second", m.Current.Metadata.ID)
	assert.NotSame(t, first, unsafe.SliceData(m.functionSpans), "the aggregate was kept across traces")
	assert.Len(t, m.functionSpans, 10)
}

// Closing the trace leaves nothing to aggregate.
func TestFunctionsSetRows_ClearedWithNoTrace(t *testing.T) {
	m := functionsModel(t, 100)

	m.Current = nil
	m.functionsSetRows()

	assert.Nil(t, m.functionSpans)
	assert.Nil(t, m.functionSpansTrace)
	assert.Nil(t, m.functionVisible)
	assert.Equal(t, 0, m.functions.Len())
}

// BenchmarkFunctionsSetRows is the cost of one filter keystroke on an open
// trace.
//
// "cached" is what a keystroke costs: the rows are rebuilt from the aggregate
// the open trace already has. "reaggregated" is what it cost when every
// rebuild segmented the calls again, which is the difference the cache makes.
func BenchmarkFunctionsSetRows(b *testing.B) {
	for _, calls := range []int{10_000, 100_000} {
		for _, cached := range []bool{true, false} {
			name := "cached"
			if !cached {
				name = "reaggregated"
			}

			b.Run(fmt.Sprintf("%s/calls=%d", name, calls), func(b *testing.B) {
				m := NewModel("/tmp/compass.so", 10, 10)
				m.Init()
				m.Height, m.Width = 40, 160
				m.updateTrace(functionsTrace("first", calls))
				m.updateKeyEnter()
				m.storeFilter(PageFunctions, "loadMultiple")

				b.ReportAllocs()
				b.ResetTimer()

				for b.Loop() {
					if !cached {
						m.functionsInvalidateSpans()
					}

					m.functionsSetRows()
				}

				b.StopTimer()
				b.ReportMetric(float64(len(m.functionSpans)), "spans")
			})
		}
	}
}
