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

// functionsTrace of a request carrying spans, spread over a one second
// request so that they sit at different points in it.
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
		Spans: make([]trace.Span, 0, calls),
		Calls: int64(calls),
	}

	// One span per call, which is the worst case the page can be handed: a
	// request whose calls all landed in different slices of it.
	for i := range calls {
		fullTrace.Spans = append(fullTrace.Spans, trace.Span{
			Name:    fmt.Sprintf("Drupal\\Core\\Entity\\Sql\\SqlContentEntityStorage%d::loadMultiple", i%400),
			Offset:  time.Duration(i) * time.Microsecond % time.Second,
			Elapsed: time.Duration(i%500) * time.Microsecond,
			Total:   time.Duration(i%500) * time.Microsecond,
			Calls:   1,
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

// The ordered spans depend on the trace alone, so rebuilding the rows for a
// different filter must reuse them rather than sort them again.
func TestFunctionsSetRows_ReusesTheAggregateWhileTheTraceIsOpen(t *testing.T) {
	m := functionsModel(t, 2_000)

	first := unsafe.SliceData(m.functionSpans)
	spans := len(m.functionSpans)

	m.storeFilter(PageFunctions, "loadMultiple")
	m.functionsSetRows()

	assert.Same(t, first, unsafe.SliceData(m.functionSpans), "filtering reordered the spans again")
	assert.Len(t, m.functionSpans, spans, "filtering changed the spans")
	assert.NotEmpty(t, m.functionVisible, "the filter matched nothing, so it proves nothing")

	// A resize rebuilds the rows, which carry their own widths, but not the
	// aggregate they are built from.
	m.functionsSetRows()

	assert.Same(t, first, unsafe.SliceData(m.functionSpans), "a rebuild reordered the spans again")
}

// Opening another trace has to replace the spans, or the page would show the
// previous request's calls.
func TestFunctionsSetRows_RecomputesForAnotherTrace(t *testing.T) {
	m := functionsModel(t, 2_000)

	first := unsafe.SliceData(m.functionSpans)

	m.updateKeyEsc()
	m.updateTrace(functionsTrace("second", 10))
	m.search.SetCursor(0)
	m.updateKeyEnter()

	require.Equal(t, "second", m.Current.Metadata.ID)
	assert.NotSame(t, first, unsafe.SliceData(m.functionSpans), "the spans were kept across traces")
	assert.Len(t, m.functionSpans, 10)
}

// Closing the trace leaves nothing to show.
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
// "cached" is what a keystroke costs: the rows are rebuilt from the spans the
// open trace already has in order. "resorted" is what it costs when every
// rebuild orders them again, which is the difference the cache makes.
func BenchmarkFunctionsSetRows(b *testing.B) {
	for _, calls := range []int{10_000, 100_000} {
		for _, cached := range []bool{true, false} {
			name := "cached"
			if !cached {
				name = "resorted"
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
