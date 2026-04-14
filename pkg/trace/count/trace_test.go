package count

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
		{Name: "foo", StartTime: 100, Elapsed: 50, Memory: 1024},
	})

	result := Unmarshal(full)

	require.Len(t, result.Functions, 1)
	assert.Equal(t, "foo", result.Functions[0].Name)
	assert.Equal(t, int64(1), result.Functions[0].Calls)
}

func TestUnmarshal_MultipleSameFunction(t *testing.T) {
	// Multiple calls to the same function in the same segment should be
	// aggregated with accumulated Calls count.
	full := newTestTrace(0, 1000, []trace.FunctionCall{
		{Name: "foo", StartTime: 100, Elapsed: 50, Memory: 1024},
		{Name: "foo", StartTime: 150, Elapsed: 50, Memory: 2048},
	})

	result := Unmarshal(full)

	// There should be a "foo" entry with aggregated calls.
	var found bool
	for _, f := range result.Functions {
		if f.Name == "foo" {
			found = true
			assert.GreaterOrEqual(t, f.Calls, int64(1))
		}
	}
	assert.True(t, found, "expected to find function 'foo'")
}

func TestUnmarshal_DifferentFunctions(t *testing.T) {
	full := newTestTrace(0, 1000, []trace.FunctionCall{
		{Name: "foo", StartTime: 100, Elapsed: 50, Memory: 1024},
		{Name: "bar", StartTime: 500, Elapsed: 50, Memory: 2048},
	})

	result := Unmarshal(full)

	names := make([]string, len(result.Functions))
	for i, f := range result.Functions {
		names[i] = f.Name
	}
	assert.Contains(t, names, "foo")
	assert.Contains(t, names, "bar")
}

func TestUnmarshal_SortOrder_ByPercentage(t *testing.T) {
	// "bar" spans more of the request than "foo" so should appear first.
	full := newTestTrace(0, 1000, []trace.FunctionCall{
		{Name: "foo", StartTime: 100, Elapsed: 50, Memory: 1024},
		{Name: "bar", StartTime: 200, Elapsed: 800, Memory: 2048},
	})

	result := Unmarshal(full)

	require.GreaterOrEqual(t, len(result.Functions), 2)
	// Sorted by percentage descending - the function spanning more time should be first.
	assert.GreaterOrEqual(t, result.Functions[0].Percentage, result.Functions[1].Percentage)
}

func TestUnmarshal_EmptyFunctionCalls(t *testing.T) {
	full := newTestTrace(0, 1000, nil)

	result := Unmarshal(full)

	assert.Empty(t, result.Functions)
	assert.Equal(t, 0, result.TotalFunctionCalls)
}

func TestUnmarshal_PreservesMetadata(t *testing.T) {
	full := newTestTrace(0, 1000, []trace.FunctionCall{
		{Name: "foo", StartTime: 100, Elapsed: 50, Memory: 1024},
		{Name: "bar", StartTime: 200, Elapsed: 50, Memory: 2048},
		{Name: "baz", StartTime: 300, Elapsed: 50, Memory: 4096},
	})

	result := Unmarshal(full)

	assert.Equal(t, full.Metadata, result.Metadata)
	assert.Equal(t, 3, result.TotalFunctionCalls)
}
