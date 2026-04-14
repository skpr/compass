package segmented

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skpr/compass/pkg/trace"
)

func newTestTrace(startTime, endTime int64, calls []trace.FunctionCall) trace.Trace {
	return trace.Trace{
		Metadata: trace.Metadata{
			Source:    trace.SourceHTTP,
			ID:        "test-id",
			StartTime: startTime,
			EndTime:   endTime,
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
		{Name: "foo", StartTime: 100, Elapsed: 200, Memory: 1024},
	})

	result := Unmarshal(full, 10)

	require.Len(t, result.Spans, 1)
	assert.Equal(t, "foo", result.Spans[0].Name)
	assert.Equal(t, int64(100), result.Spans[0].StartTime)
	assert.Equal(t, int64(100), result.Spans[0].Start)
	assert.Equal(t, int64(200), result.Spans[0].Length)
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
		{Name: "foo", StartTime: 150, Elapsed: 50, Memory: 2048},
		{Name: "foo", StartTime: 100, Elapsed: 50, Memory: 1024},
	})

	result := Unmarshal(full, 10)

	require.Len(t, result.Spans, 1)
	assert.Equal(t, 2, result.Spans[0].TotalFunctionCalls)
	// Should keep the earliest StartTime.
	assert.Equal(t, int64(100), result.Spans[0].StartTime)
	// Should keep the highest MaxMemory.
	assert.Equal(t, int64(2048), result.Spans[0].MaxMemory)
}

func TestUnmarshal_MultipleFunctions_DifferentSegments(t *testing.T) {
	// Segment length = 1000/10 = 100.
	// First call: keyStart = (100-0)/100 = 1 → key "foo-1-1".
	// Second call: keyStart = (500-0)/100 = 5 → key "foo-5-1".
	full := newTestTrace(0, 1000, []trace.FunctionCall{
		{Name: "foo", StartTime: 100, Elapsed: 50, Memory: 1024},
		{Name: "foo", StartTime: 500, Elapsed: 50, Memory: 2048},
	})

	result := Unmarshal(full, 10)

	assert.Len(t, result.Spans, 2)
}

func TestUnmarshal_DifferentFunctions(t *testing.T) {
	full := newTestTrace(0, 1000, []trace.FunctionCall{
		{Name: "foo", StartTime: 100, Elapsed: 50, Memory: 1024},
		{Name: "bar", StartTime: 100, Elapsed: 50, Memory: 2048},
	})

	result := Unmarshal(full, 10)

	assert.Len(t, result.Spans, 2)

	names := []string{result.Spans[0].Name, result.Spans[1].Name}
	assert.Contains(t, names, "foo")
	assert.Contains(t, names, "bar")
}

func TestUnmarshal_SortOrder(t *testing.T) {
	full := newTestTrace(0, 1000, []trace.FunctionCall{
		{Name: "beta", StartTime: 500, Elapsed: 50, Memory: 1024},
		{Name: "alpha", StartTime: 500, Elapsed: 50, Memory: 1024},
		{Name: "alpha", StartTime: 100, Elapsed: 50, Memory: 1024},
	})

	result := Unmarshal(full, 10)

	require.Len(t, result.Spans, 3)

	// First sorted by StartTime, then by Name.
	assert.Equal(t, "alpha", result.Spans[0].Name)
	assert.Equal(t, int64(100), result.Spans[0].StartTime)
	assert.Equal(t, "alpha", result.Spans[1].Name)
	assert.Equal(t, int64(500), result.Spans[1].StartTime)
	assert.Equal(t, "beta", result.Spans[2].Name)
	assert.Equal(t, int64(500), result.Spans[2].StartTime)
}

func TestUnmarshal_EmptyFunctionCalls(t *testing.T) {
	full := newTestTrace(0, 1000, nil)

	result := Unmarshal(full, 10)

	assert.Empty(t, result.Spans)
	assert.Equal(t, 0, result.TotalFunctionCalls)
}

func TestUnmarshal_PreservesMetadata(t *testing.T) {
	full := newTestTrace(0, 1000, []trace.FunctionCall{
		{Name: "foo", StartTime: 100, Elapsed: 50, Memory: 1024},
	})

	result := Unmarshal(full, 10)

	assert.Equal(t, full.Metadata, result.Metadata)
	assert.Equal(t, int64(10), result.Segments)
}

func TestUnmarshal_TotalFunctionCalls(t *testing.T) {
	full := newTestTrace(0, 1000, []trace.FunctionCall{
		{Name: "foo", StartTime: 100, Elapsed: 50, Memory: 1024},
		{Name: "bar", StartTime: 200, Elapsed: 50, Memory: 1024},
		{Name: "baz", StartTime: 300, Elapsed: 50, Memory: 1024},
	})

	result := Unmarshal(full, 10)

	assert.Equal(t, 3, result.TotalFunctionCalls)
}

func TestUnmarshal_MinKeyLength(t *testing.T) {
	// Segment length = 1000/10 = 100. Elapsed=10, 10/100=0 → clamped to 1.
	full := newTestTrace(0, 1000, []trace.FunctionCall{
		{Name: "foo", StartTime: 100, Elapsed: 10, Memory: 1024},
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
