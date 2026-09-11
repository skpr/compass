package fpm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skpr/compass/pkg/tracer/spans"
)

// A trace carries a bounded number of spans, and what it could not represent
// is reported rather than silently missing.
func TestHandler_SpansAreBounded(t *testing.T) {
	sink := &mockSink{}
	options := testOptions()
	options.Spans.Max = 2
	h, err := NewHandler(sink, options)
	require.NoError(t, err)

	id := makeRequestID("bounded")
	require.NoError(t, h.Handle(t.Context(), bpfEvent{Type: EventRequestInit, RequestId: id, Timestamp: 100}))

	// Four distinct functions, so the last two have nowhere to go.
	for i, memory := range []uint64{10, 20, 30, 40} {
		require.NoError(t, h.Handle(t.Context(), bpfEvent{
			Type: EventFunction, RequestId: id,
			FunctionName: makeFunctionName(string(rune('a' + i))),
			Timestamp:    uint64(200 + i), Elapsed: 1, Memory: memory,
		}))
	}

	stored, found := h.storage.Get(id, 0)
	require.True(t, found)
	assert.Len(t, stored.spans.Spans(), 2)

	require.NoError(t, h.Handle(t.Context(), bpfEvent{Type: EventRequestShutdown, RequestId: id, Timestamp: 300}))

	traces := sink.Traces()
	require.Len(t, traces, 1)
	assert.Len(t, traces[0].Spans, 2)
	assert.Equal(t, int64(4), traces[0].Calls)
	assert.Equal(t, int64(2), traces[0].CallsDropped)
	// Memory is not a sample: the calls no span represents still reported it.
	assert.Equal(t, int64(40), traces[0].ResourceUtilisation.MaxMemory)
}

// The reason a trace can be bounded at all: a function called a hundred
// thousand times within one slice of a request is one span, and the count of
// what it did is exact.
func TestHandler_AHotFunctionIsOneSpan(t *testing.T) {
	const calls = 100_000

	sink := &mockSink{}
	h, err := NewHandler(sink, testOptions())
	require.NoError(t, err)

	id := makeRequestID("hot")
	require.NoError(t, h.Handle(t.Context(), bpfEvent{Type: EventRequestInit, RequestId: id, Timestamp: 100}))

	for i := range calls {
		require.NoError(t, h.Handle(t.Context(), bpfEvent{
			Type: EventFunction, RequestId: id,
			FunctionName: makeFunctionName("hot_path"),
			Timestamp:    uint64(200 + i), Elapsed: 1, Memory: uint64(i),
		}))
	}

	require.NoError(t, h.Handle(t.Context(), bpfEvent{Type: EventRequestShutdown, RequestId: id, Timestamp: uint64(200 + calls)}))

	traces := sink.Traces()
	require.Len(t, traces, 1)
	require.Len(t, traces[0].Spans, 1)
	assert.Equal(t, int64(calls), traces[0].Spans[0].Calls)
	assert.Equal(t, int64(calls), traces[0].Calls)
	assert.Zero(t, traces[0].CallsDropped)
	assert.Equal(t, int64(calls-1), traces[0].ResourceUtilisation.MaxMemory)
}

// A function called throughout a request is a span per slice of it, which is
// what the timeline is built from, and that is what the bound applies to.
func TestHandler_SpansStayBoundedAcrossALongRequest(t *testing.T) {
	const (
		maxSpans = 64
		calls    = 10_000
		// One bucket apart, so every call lands in its own.
		apart = int(spans.DefaultBucket)
	)

	sink := &mockSink{}
	options := testOptions()
	options.Spans.Max = maxSpans
	h, err := NewHandler(sink, options)
	require.NoError(t, err)

	id := makeRequestID("long")
	require.NoError(t, h.Handle(t.Context(), bpfEvent{Type: EventRequestInit, RequestId: id, Timestamp: 100}))

	for i := range calls {
		require.NoError(t, h.Handle(t.Context(), bpfEvent{
			Type: EventFunction, RequestId: id,
			FunctionName: makeFunctionName("hot_path"),
			Timestamp:    uint64(200 + i*apart), Elapsed: 1, Memory: uint64(i),
		}))
	}

	require.NoError(t, h.Handle(t.Context(), bpfEvent{Type: EventRequestShutdown, RequestId: id, Timestamp: uint64(200 + calls*apart)}))

	traces := sink.Traces()
	require.Len(t, traces, 1)
	assert.Len(t, traces[0].Spans, maxSpans)
	assert.Equal(t, int64(calls), traces[0].Calls)
	assert.Equal(t, int64(calls-maxSpans), traces[0].CallsDropped)
}
