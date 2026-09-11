// Package segmented for storing tracing data in segments.
package segmented

import (
	"sort"
	"time"

	"github.com/skpr/compass/pkg/trace"
)

// maxPrealloc bounds the map preallocation below, so that a trace which
// carries a great many calls does not reserve a map sized for all of them
// before finding out how few distinct spans they collapse into.
const maxPrealloc = 4096

// key identifies the span a function call is aggregated into: its name, and the
// segment its start and its length fall in.
//
// A struct rather than a formatted string. The name is the only variable length
// part of it, and formatting one costs two allocations per function call, which
// on a large trace is most of what this function does.
type key struct {
	name   string
	start  time.Duration
	length time.Duration
}

// Unmarshal a full trace into a segmented trace.
func Unmarshal(fullTrace trace.Trace, segments int64) Trace {
	if segments < 1 {
		segments = 1
	}

	// A request shorter than the number of segments would give every span a
	// segment length of zero, and the bucketing below divides by it.
	segmentLength := max(fullTrace.Metadata.ExecutionTime()/time.Duration(segments), 1)

	// Sized for the calls it is about to bucket. Aggregation only shrinks that,
	// but growing the map is a per-call cost on the largest traces, which are
	// exactly the ones this has to stay cheap for.
	spans := make(map[key]Span, min(len(fullTrace.FunctionCalls), maxPrealloc))

	for _, call := range fullTrace.FunctionCalls {
		span := Span{
			Name:               call.Name,
			Offset:             call.Offset,
			Length:             call.Elapsed,
			TotalFunctionCalls: 1,
			MaxMemory:          call.Memory,
		}

		var (
			keyStart  = call.Offset / segmentLength
			keyLength = span.Length / segmentLength
		)

		if keyLength == 0 {
			keyLength = 1
		}

		key := key{name: span.Name, start: keyStart, length: keyLength}

		if val, ok := spans[key]; ok {
			span.TotalFunctionCalls = val.TotalFunctionCalls + 1

			if span.Offset > val.Offset {
				span.Offset = val.Offset
			}

			if span.MaxMemory < val.MaxMemory {
				span.MaxMemory = val.MaxMemory
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
		if segmentedTrace.Spans[i].Offset != segmentedTrace.Spans[j].Offset {
			return segmentedTrace.Spans[i].Offset < segmentedTrace.Spans[j].Offset
		}

		if segmentedTrace.Spans[i].Name != segmentedTrace.Spans[j].Name {
			return segmentedTrace.Spans[i].Name < segmentedTrace.Spans[j].Name
		}

		return segmentedTrace.Spans[i].Length < segmentedTrace.Spans[j].Length
	})

	return segmentedTrace
}
