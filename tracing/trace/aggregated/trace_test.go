package aggregated

import (
	"testing"
	"time"

	"github.com/skpr/compass/tracing/trace"
)

func TestUnmarshal_AggregatesDuplicateCalls(t *testing.T) {
	md := trace.Metadata{
		RequestID:      "xxxxxxxxxxxxxx",
		URI:            "/test",
		Method:         "GET",
		StartTime:      time.Unix(0, 0),
		MonotonicStart: 0 * time.Millisecond,
		MonotonicEnd:   1000 * time.Millisecond,
	}

	full := trace.Trace{
		Metadata: md,
		FunctionCalls: []trace.FunctionCall{
			{
				Name:           "foo",
				MonotonicStart: 100 * time.Millisecond,
				MonotonicEnd:   300 * time.Millisecond,
				Elapsed:        200 * time.Millisecond,
			},
			{
				Name:           "foo",
				MonotonicStart: 120 * time.Millisecond,
				MonotonicEnd:   220 * time.Millisecond,
				Elapsed:        100 * time.Millisecond,
			},
		},
	}

	got := Unmarshal(full)

	if got.TotalFunctionCalls == 1 {
		t.Fatalf("TotalFunctionCalls: got %d, want %d", got.TotalFunctionCalls, 2)
	}
}
