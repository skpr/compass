// Package segmented for storing tracing data in segments.
package segmented

import (
	"fmt"
	"sort"

	"github.com/skpr/compass/pkg/trace"
)

// Unmarshal a full trace into a segmented trace.
func Unmarshal(fullTrace trace.Trace, segments int64) Trace {
	if segments < 1 {
		segments = 1
	}

	// A request shorter than the number of segments would give every span a
	// segment length of zero, and the bucketing below divides by it.
	segmentLength := max((fullTrace.Metadata.EndTime-fullTrace.Metadata.StartTime)/segments, 1)

	selfTimes := SelfTime(fullTrace.FunctionCalls)

	spans := make(map[string]Span)

	for i, call := range fullTrace.FunctionCalls {
		span := Span{
			Name:               call.Name,
			StartTime:          call.StartTime,
			Start:              call.StartTime - fullTrace.Metadata.StartTime,
			Length:             call.Elapsed,
			TotalFunctionCalls: 1,
			MaxMemory:          call.Memory,
			SelfTime:           selfTimes[i],
		}

		var (
			keyStart  = (call.StartTime - fullTrace.Metadata.StartTime) / segmentLength
			keyLength = span.Length / segmentLength
		)

		if keyLength == 0 {
			keyLength = 1
		}

		key := fmt.Sprintf("%s-%d-%d", span.Name, keyStart, keyLength)

		if val, ok := spans[key]; ok {
			span.TotalFunctionCalls = val.TotalFunctionCalls + 1

			if span.StartTime > val.StartTime {
				span.StartTime = val.StartTime
			}

			if span.MaxMemory < val.MaxMemory {
				span.MaxMemory = val.MaxMemory
			}

			// Self time is the one field which accumulates rather than being
			// taken from one of the merged calls: two calls of the same
			// function each did their own work.
			span.SelfTime += val.SelfTime

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
