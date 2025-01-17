// Package segmented for storing tracing data in segments.
package segmented

import (
	"fmt"
	"sort"

	"github.com/skpr/compass/trace"
)

// Unmarshal a full trace into a segmented trace.
func Unmarshal(fullTrace trace.Trace, segments int) Trace {
	spans := make(map[string]Span)

	for _, call := range fullTrace.FunctionCalls {
		span := Span{
			Name:               call.Name,
			StartTime:          call.StartTime,
			Start:              getSegmentStart(fullTrace.Metadata.StartTime, call.StartTime, fullTrace.Metadata.ExecutionTime, float64(segments)),
			Length:             getSegmentLength(fullTrace.Metadata.ExecutionTime, (call.EndTime-call.StartTime)/1000, float64(segments)),
			TotalFunctionCalls: 1,
		}

		key := fmt.Sprintf("%s-%d-%d", span.Name, span.Start, span.Length)

		if val, ok := spans[key]; ok {
			span.TotalFunctionCalls = val.TotalFunctionCalls + 1

			if span.StartTime > val.StartTime {
				span.StartTime = val.StartTime
			}

			spans[key] = span
			continue
		}

		spans[key] = span
	}

	segmentedTrace := Trace{
		Metadata:           fullTrace.Metadata,
		Segments:           segments,
		TotalFunctionCalls: len(fullTrace.FunctionCalls),
	}

	for _, span := range spans {
		segmentedTrace.Spans = append(segmentedTrace.Spans, span)
	}

	// We also need to sort these now that all the spans have gone through a map which does not have ordering.
	sort.Slice(segmentedTrace.Spans, func(i, j int) bool {
		if segmentedTrace.Spans[i].StartTime != segmentedTrace.Spans[j].StartTime {
			return segmentedTrace.Spans[i].StartTime < segmentedTrace.Spans[j].StartTime
		}

		if segmentedTrace.Spans[i].Name != segmentedTrace.Spans[j].Name {
			return segmentedTrace.Spans[i].Name < segmentedTrace.Spans[j].Name
		}

		return segmentedTrace.Spans[i].Length < segmentedTrace.Spans[j].Length
	})

	return segmentedTrace
}
