// Package aggregated for storing aggregated tracing data.
package aggregated

import (
	"fmt"
	"sort"
	"time"

	"github.com/skpr/compass/tracing/trace"
)

// Unmarshal a full trace into a segmented trace.
func Unmarshal(fullTrace trace.Trace) Trace {
	// We are using 5% buffer on the before/after function call for our rollup.
	// We can make this configurable in a future release.
	segmentDuration := time.Duration(float64(fullTrace.Metadata.ExecutionTime()) * 0.05)

	spans := make(map[string]Span)

	for _, call := range fullTrace.FunctionCalls {
		var (
			start   = fullTrace.Metadata.ExecutionTime() / segmentDuration
			elapsed = call.Elapsed / segmentDuration
		)

		key := fmt.Sprintf("%s-%d-%d", call.Name, start.Nanoseconds(), elapsed.Nanoseconds())

		span := Span{
			Name:    call.Name,
			Start:   call.MonotonicStart - fullTrace.Metadata.MonotonicStart,
			Elapsed: call.Elapsed,
			Calls:   1,
		}

		if val, ok := spans[key]; ok {
			span.Calls++

			if span.Start < val.Start {
				span.Start = val.Start
			}

			if span.End > val.End {
				span.End = val.End
			}

			if span.Elapsed > val.Elapsed {
				span.Elapsed = val.Elapsed
			}

			spans[key] = span
			continue
		}

		spans[key] = span
	}

	aggregatedTrace := Trace{
		Metadata:           fullTrace.Metadata,
		TotalFunctionCalls: len(fullTrace.FunctionCalls),
	}

	for _, span := range spans {
		aggregatedTrace.Spans = append(aggregatedTrace.Spans, span)
	}

	// We also need to sort these now that all the spans have gone through a map which does not have ordering.
	sort.Slice(aggregatedTrace.Spans, func(i, j int) bool {
		if aggregatedTrace.Spans[i].Start != aggregatedTrace.Spans[j].Start {
			return aggregatedTrace.Spans[i].Start < aggregatedTrace.Spans[j].Start
		}

		if aggregatedTrace.Spans[i].Name != aggregatedTrace.Spans[j].Name {
			return aggregatedTrace.Spans[i].Name < aggregatedTrace.Spans[j].Name
		}

		return aggregatedTrace.Spans[i].Elapsed < aggregatedTrace.Spans[j].Elapsed
	})

	return aggregatedTrace
}
