package segmented_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/skpr/compass/pkg/trace"
	"github.com/skpr/compass/pkg/trace/segmented"
)

// benchmarkSymbols is roughly how many distinct functions a Drupal request
// calls above the extension's threshold. The call count varies below; this does
// not, because it is what the aggregate collapses to.
const benchmarkSymbols = 400

// benchmarkTrace of a one second request with calls spread across it.
func benchmarkTrace(calls int) trace.Trace {
	start := time.Now()

	names := make([]string, benchmarkSymbols)
	for i := range names {
		names[i] = fmt.Sprintf("Drupal\\Core\\Entity\\Sql\\SqlContentEntityStorage%d::loadMultiple", i)
	}

	fullTrace := trace.Trace{
		Metadata: trace.Metadata{
			ID:        "0123456789abcdef",
			Source:    trace.SourceHTTP,
			Runtime:   trace.RuntimePHP,
			HTTP:      trace.MetadataHTTP{Method: "GET", URI: "/node/1234"},
			StartTime: start,
			EndTime:   start.Add(time.Second),
		},
		FunctionCalls: make([]trace.FunctionCall, 0, calls),
	}

	for i := range calls {
		fullTrace.FunctionCalls = append(fullTrace.FunctionCalls, trace.FunctionCall{
			Name:    names[i%len(names)],
			Offset:  time.Duration(i) * time.Microsecond % time.Second,
			Elapsed: time.Duration(i%500) * time.Microsecond,
			Memory:  int64(i) * 128,
		})
	}

	return fullTrace
}

// BenchmarkUnmarshal aggregates traces from the size the sidecar retains today
// up to the size a large Drupal request actually produces.
func BenchmarkUnmarshal(b *testing.B) {
	for _, calls := range []int{10_000, 100_000, 1_000_000} {
		fullTrace := benchmarkTrace(calls)

		b.Run(fmt.Sprintf("calls=%d", calls), func(b *testing.B) {
			b.ReportAllocs()

			var segmentedTrace segmented.Trace

			for b.Loop() {
				segmentedTrace = segmented.Unmarshal(fullTrace, 100)
			}

			// Reported so that a change which makes this cheaper by aggregating
			// more coarsely is visible as such rather than as a win.
			b.ReportMetric(float64(len(segmentedTrace.Spans)), "spans")
		})
	}
}
