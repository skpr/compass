package spans

import (
	"fmt"
	"testing"
	"time"
	"unsafe"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skpr/compass/pkg/trace"
)

func name(value string) []byte {
	return append([]byte(value), 0)
}

// Calls of the same function in the same slice of the request are one span,
// which is what keeps a trace of a million calls small enough to carry.
func TestBuilder_AggregatesRepeatedCalls(t *testing.T) {
	builder := New(Options{Max: 100, Bucket: time.Millisecond}, RuntimePHPFPM).Request()

	builder.Add(name("loadMultiple"), 2*time.Millisecond, 300*time.Microsecond, 1000)
	builder.Add(name("loadMultiple"), 2*time.Millisecond+500*time.Microsecond, 900*time.Microsecond, 3000)
	builder.Add(name("loadMultiple"), 2*time.Millisecond+900*time.Microsecond, 100*time.Microsecond, 2000)

	var tr trace.Trace
	builder.Finish(&tr)

	require.Len(t, tr.Spans, 1)
	assert.Equal(t, "loadMultiple", tr.Spans[0].Name)
	assert.Equal(t, int64(3), tr.Spans[0].Calls)
	// The earliest call in the span places it, the longest sizes it, and the
	// three together are what it cost.
	assert.Equal(t, 2*time.Millisecond, tr.Spans[0].Offset)
	assert.Equal(t, 900*time.Microsecond, tr.Spans[0].Elapsed)
	assert.Equal(t, 1300*time.Microsecond, tr.Spans[0].Total)
	assert.Equal(t, int64(3000), tr.Spans[0].Memory)
	assert.Equal(t, int64(3), tr.Calls)
	assert.Equal(t, int64(3000), tr.ResourceUtilisation.MaxMemory)
}

// When in the request a call happened is what the timeline shows, so the same
// function called at two points in a request is two spans.
func TestBuilder_SeparatesCallsByWhenTheyRan(t *testing.T) {
	builder := New(Options{Max: 100, Bucket: time.Millisecond}, RuntimePHPFPM).Request()

	builder.Add(name("render"), 1*time.Millisecond, time.Microsecond, 0)
	builder.Add(name("render"), 800*time.Millisecond, time.Microsecond, 0)

	var tr trace.Trace
	builder.Finish(&tr)

	require.Len(t, tr.Spans, 2)
	assert.Equal(t, 1*time.Millisecond, tr.Spans[0].Offset)
	assert.Equal(t, 800*time.Millisecond, tr.Spans[1].Offset)
}

func TestBuilder_SeparatesDistinctFunctions(t *testing.T) {
	builder := New(Options{Max: 100, Bucket: time.Millisecond}, RuntimePHPFPM).Request()

	builder.Add(name("render"), 0, time.Microsecond, 0)
	builder.Add(name("loadMultiple"), 0, time.Microsecond, 0)

	var tr trace.Trace
	builder.Finish(&tr)

	assert.Len(t, tr.Spans, 2)
}

// A trace is bounded, and what it could not represent is reported rather than
// silently missing: the call count stays exact and the dropped count says how
// much of it no span covers.
func TestBuilder_BoundsSpansAndReportsWhatItDropped(t *testing.T) {
	// The counter is process-global, so this is a delta.
	dropped := testutil.ToFloat64(metricCallsDropped.WithLabelValues(string(RuntimePHPFPM)))

	builder := New(Options{Max: 2, Bucket: time.Millisecond}, RuntimePHPFPM).Request()

	// Three distinct functions, so the third has nowhere to go.
	builder.Add(name("first"), 0, time.Microsecond, 10)
	builder.Add(name("second"), 0, time.Microsecond, 20)
	builder.Add(name("third"), 0, time.Microsecond, 30)
	builder.Add(name("third"), 0, time.Microsecond, 40)

	var tr trace.Trace
	builder.Finish(&tr)

	assert.Len(t, tr.Spans, 2)
	assert.Equal(t, int64(4), tr.Calls)
	assert.Equal(t, int64(2), tr.CallsDropped)
	// Memory is not a sample: the calls no span represents still reported it.
	assert.Equal(t, int64(40), tr.ResourceUtilisation.MaxMemory)

	assert.Equal(t, dropped+2, testutil.ToFloat64(metricCallsDropped.WithLabelValues(string(RuntimePHPFPM))))
}

// A function called twice is one string, not two: a request repeats a few
// hundred names thousands of times.
func TestAggregator_ReusesNames(t *testing.T) {
	aggregator := New(Options{Max: 100, Bucket: time.Millisecond}, RuntimePHPFPM)

	first := aggregator.Request()
	first.Add(name("Drupal\\Core\\Entity::loadMultiple"), 0, time.Microsecond, 0)

	// A separate request, so a shared string can only come from the
	// aggregator holding on to the first one.
	second := aggregator.Request()
	second.Add(name("Drupal\\Core\\Entity::loadMultiple"), 0, time.Microsecond, 0)

	assert.Same(t,
		unsafe.StringData(first.Spans()[0].Name),
		unsafe.StringData(second.Spans()[0].Name),
		"the same function was named twice",
	)
}

// The names come from the application, so a workload which generates them at
// runtime must not grow the aggregator without bound. Past the bound the
// names are still reported.
func TestAggregator_BoundsTheNamesItHolds(t *testing.T) {
	aggregator := New(Options{Max: MaxNames * 2, Bucket: time.Millisecond}, RuntimePHPFPM)
	builder := aggregator.Request()

	for i := range MaxNames + 10 {
		builder.Add(name(fmt.Sprintf("generated_%d", i)), 0, time.Microsecond, 0)
	}

	assert.Len(t, aggregator.names, MaxNames)
	require.Len(t, builder.Spans(), MaxNames+10)
	assert.Equal(t, fmt.Sprintf("generated_%d", MaxNames+9), builder.Spans()[MaxNames+9].Name)
}

func TestNew_Defaults(t *testing.T) {
	aggregator := New(Options{}, RuntimePHPFPM)

	assert.Equal(t, DefaultMax, aggregator.Max())
	assert.Equal(t, DefaultBucket, aggregator.Bucket())
}

func TestNew_UnknownRuntime(t *testing.T) {
	assert.Panics(t, func() { New(Options{Max: 10, Bucket: time.Millisecond}, Runtime("unknown")) })
}

// BenchmarkAdd is the per-function-call cost of aggregating, which every call
// of a million-call request pays.
func BenchmarkAdd(b *testing.B) {
	const symbols = 400

	names := make([][]byte, symbols)
	for i := range names {
		names[i] = name(fmt.Sprintf("Drupal\\Core\\Entity\\Sql\\SqlContentEntityStorage%d::loadMultiple", i))
	}

	aggregator := New(Options{}, RuntimePHPFPM)
	builder := aggregator.Request()

	b.ReportAllocs()

	for i := 0; b.Loop(); i++ {
		builder.Add(names[i%symbols], time.Duration(i%1000)*time.Millisecond, time.Microsecond, int64(i))
	}

	b.StopTimer()
	b.ReportMetric(float64(len(builder.Spans())), "spans")
}
