package segmented

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skpr/compass/pkg/trace"
)

// origin the test traces are measured from.
//
// Timestamps are instants now, and the offsets below are still counted in
// nanoseconds from the start of the request, so they read the way they did when
// a timestamp was a count.
var origin = time.Unix(1700000000, 0)

// at an offset into the test request.
func at(offset time.Duration) time.Time {
	return origin.Add(offset)
}

func newTestTrace(startTime, endTime time.Duration, calls []trace.FunctionCall) trace.Trace {
	return trace.Trace{
		Metadata: trace.Metadata{
			Source:    trace.SourceHTTP,
			ID:        "test-id",
			StartTime: at(startTime),
			EndTime:   at(endTime),
			HTTP: trace.MetadataHTTP{
				Method: "GET",
				URI:    "/test",
			},
		},
		FunctionCalls: calls,
	}
}

func TestUnmarshal_SingleFunction(t *testing.T) {
	full := newTestTrace(0, 1000, []trace.FunctionCall{
		{Name: "foo", Offset: 100, Elapsed: 200, Memory: 1024},
	})

	result := Unmarshal(full, 10)

	require.Len(t, result.Spans, 1)
	assert.Equal(t, "foo", result.Spans[0].Name)
	assert.Equal(t, 100*time.Nanosecond, result.Spans[0].Offset)
	assert.Equal(t, 100*time.Nanosecond, result.Spans[0].Offset)
	assert.Equal(t, 200*time.Nanosecond, result.Spans[0].Length)
	assert.Equal(t, int64(1024), result.Spans[0].MaxMemory)
	assert.Equal(t, 1, result.Spans[0].TotalFunctionCalls)
}

func TestUnmarshal_MultipleFunctions_SameSegment(t *testing.T) {
	// Two calls to "foo" that fall into the same segment key.
	// Segment length = 1000/10 = 100.
	// Both start at 100, keyStart = (100-0)/100 = 1.
	// Both have elapsed 50, keyLength = 50/100 = 0 → clamped to 1.
	// Same key: "foo-1-1".
	full := newTestTrace(0, 1000, []trace.FunctionCall{
		{Name: "foo", Offset: 150, Elapsed: 50, Memory: 2048},
		{Name: "foo", Offset: 100, Elapsed: 50, Memory: 1024},
	})

	result := Unmarshal(full, 10)

	require.Len(t, result.Spans, 1)
	assert.Equal(t, 2, result.Spans[0].TotalFunctionCalls)
	// Should keep the earliest offset.
	assert.Equal(t, 100*time.Nanosecond, result.Spans[0].Offset)
	// Should keep the highest MaxMemory.
	assert.Equal(t, int64(2048), result.Spans[0].MaxMemory)
}

func TestUnmarshal_MultipleFunctions_DifferentSegments(t *testing.T) {
	// Segment length = 1000/10 = 100.
	// First call: keyStart = (100-0)/100 = 1 → key "foo-1-1".
	// Second call: keyStart = (500-0)/100 = 5 → key "foo-5-1".
	full := newTestTrace(0, 1000, []trace.FunctionCall{
		{Name: "foo", Offset: 100, Elapsed: 50, Memory: 1024},
		{Name: "foo", Offset: 500, Elapsed: 50, Memory: 2048},
	})

	result := Unmarshal(full, 10)

	assert.Len(t, result.Spans, 2)
}

func TestUnmarshal_DifferentFunctions(t *testing.T) {
	full := newTestTrace(0, 1000, []trace.FunctionCall{
		{Name: "foo", Offset: 100, Elapsed: 50, Memory: 1024},
		{Name: "bar", Offset: 100, Elapsed: 50, Memory: 2048},
	})

	result := Unmarshal(full, 10)

	assert.Len(t, result.Spans, 2)

	names := []string{result.Spans[0].Name, result.Spans[1].Name}
	assert.Contains(t, names, "foo")
	assert.Contains(t, names, "bar")
}

func TestUnmarshal_SortOrder(t *testing.T) {
	full := newTestTrace(0, 1000, []trace.FunctionCall{
		{Name: "beta", Offset: 500, Elapsed: 50, Memory: 1024},
		{Name: "alpha", Offset: 500, Elapsed: 50, Memory: 1024},
		{Name: "alpha", Offset: 100, Elapsed: 50, Memory: 1024},
	})

	result := Unmarshal(full, 10)

	require.Len(t, result.Spans, 3)

	// First sorted by Offset, then by Name.
	assert.Equal(t, "alpha", result.Spans[0].Name)
	assert.Equal(t, 100*time.Nanosecond, result.Spans[0].Offset)
	assert.Equal(t, "alpha", result.Spans[1].Name)
	assert.Equal(t, 500*time.Nanosecond, result.Spans[1].Offset)
	assert.Equal(t, "beta", result.Spans[2].Name)
	assert.Equal(t, 500*time.Nanosecond, result.Spans[2].Offset)
}

func TestUnmarshal_EmptyFunctionCalls(t *testing.T) {
	full := newTestTrace(0, 1000, nil)

	result := Unmarshal(full, 10)

	assert.Empty(t, result.Spans)
	assert.Equal(t, 0, result.TotalFunctionCalls)
}

func TestUnmarshal_PreservesMetadata(t *testing.T) {
	full := newTestTrace(0, 1000, []trace.FunctionCall{
		{Name: "foo", Offset: 100, Elapsed: 50, Memory: 1024},
	})

	result := Unmarshal(full, 10)

	assert.Equal(t, full.Metadata, result.Metadata)
	assert.Equal(t, int64(10), result.Segments)
}

func TestUnmarshal_TotalFunctionCalls(t *testing.T) {
	full := newTestTrace(0, 1000, []trace.FunctionCall{
		{Name: "foo", Offset: 100, Elapsed: 50, Memory: 1024},
		{Name: "bar", Offset: 200, Elapsed: 50, Memory: 1024},
		{Name: "baz", Offset: 300, Elapsed: 50, Memory: 1024},
	})

	result := Unmarshal(full, 10)

	assert.Equal(t, 3, result.TotalFunctionCalls)
}

func TestUnmarshal_MinKeyLength(t *testing.T) {
	// Segment length = 1000/10 = 100. Elapsed=10, 10/100=0 → clamped to 1.
	full := newTestTrace(0, 1000, []trace.FunctionCall{
		{Name: "foo", Offset: 100, Elapsed: 10, Memory: 1024},
	})

	result := Unmarshal(full, 10)

	require.Len(t, result.Spans, 1)
	// Span should still exist (keyLength clamped to 1, not 0).
	assert.Equal(t, "foo", result.Spans[0].Name)
}

func TestSpan_GetName_SingleCall(t *testing.T) {
	s := Span{
		Name:               "myFunc",
		TotalFunctionCalls: 1,
	}

	assert.Equal(t, "myFunc", s.GetName())
}

func TestSpan_GetName_MultipleCalls(t *testing.T) {
	s := Span{
		Name:               "myFunc",
		TotalFunctionCalls: 3,
	}

	assert.Equal(t, "myFunc (3)", s.GetName())
}

func TestSpan_DurationShare(t *testing.T) {
	span := Span{Length: 250 * time.Nanosecond}

	assert.InDelta(t, 0.25, span.DurationShare(1000*time.Nanosecond), 0.001)
	assert.Zero(t, span.DurationShare(0))
	assert.Zero(t, span.DurationShare(-time.Nanosecond))
}

// A request too short to have one nanosecond per segment must not divide by
// zero when calls are bucketed.
func TestUnmarshal_VeryShortTrace(t *testing.T) {
	assert.NotPanics(t, func() {
		Unmarshal(newTestTrace(0, 5, []trace.FunctionCall{
			{Name: "a", Offset: 0, Elapsed: 5},
		}), 50)
	})
}

func TestUnmarshal_ZeroDurationTrace(t *testing.T) {
	assert.NotPanics(t, func() {
		Unmarshal(newTestTrace(100, 100, []trace.FunctionCall{
			{Name: "a", Offset: 100, Elapsed: 0},
		}), 50)
	})
}
