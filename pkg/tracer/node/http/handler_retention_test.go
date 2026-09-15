package http

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skpr/compass/pkg/tracer/clock"
	"github.com/skpr/compass/pkg/tracer/spans"
)

// A trace carries a bounded number of spans, and what it could not represent
// is reported rather than silently missing.
func TestHandler_SpansAreBounded(t *testing.T) {
	sink := &readerTestSink{}
	h, err := NewHandler(sink, Options{
		Expire: time.Minute,
		Spans:  spans.Options{Max: 2},
		Clock:  clock.Monotonic{Boot: readerTestBoot},
	})
	require.NoError(t, err)

	id := readerRequestID("bounded")
	require.NoError(t, h.Handle(t.Context(), bpfEvent{Type: EventRequestInit, RequestId: id, Timestamp: 100}))

	// Four distinct functions, so the last two have nowhere to go.
	for i, memory := range []uint64{10, 20, 30, 40} {
		require.NoError(t, h.Handle(t.Context(), bpfEvent{
			Type: EventFunction, RequestId: id,
			FunctionName: readerFunctionName(string(rune('a' + i))),
			Timestamp:    uint64(200 + i), Elapsed: 1, Memory: memory,
		}))
	}

	stored, found := h.storage.Get(id, 0)
	require.True(t, found)
	assert.Len(t, stored.spans.Spans(), 2)

	require.NoError(t, h.Handle(t.Context(), bpfEvent{Type: EventRequestShutdown, RequestId: id, Timestamp: 300}))

	require.Len(t, sink.traces, 1)
	assert.Len(t, sink.traces[0].Spans, 2)
	assert.Equal(t, int64(4), sink.traces[0].Calls)
	assert.Equal(t, int64(2), sink.traces[0].CallsDropped)
	// Memory is not a sample: the calls no span represents still reported it.
	assert.Equal(t, int64(40), sink.traces[0].ResourceUtilisation.MaxMemory)
}

// A handler called repeatedly within one slice of a request is one span, not
// one record per call.
func TestHandler_AHotFunctionIsOneSpan(t *testing.T) {
	const calls = 10_000

	sink := &readerTestSink{}
	h, err := NewHandler(sink, Options{
		Expire: time.Minute,
		Clock:  clock.Monotonic{Boot: readerTestBoot},
	})
	require.NoError(t, err)

	id := readerRequestID("hot")
	require.NoError(t, h.Handle(t.Context(), bpfEvent{Type: EventRequestInit, RequestId: id, Timestamp: 100}))

	for i := range calls {
		require.NoError(t, h.Handle(t.Context(), bpfEvent{
			Type: EventFunction, RequestId: id,
			FunctionName: readerFunctionName("hot_path"),
			Timestamp:    uint64(200 + i), Elapsed: 1, Memory: uint64(i),
		}))
	}

	require.NoError(t, h.Handle(t.Context(), bpfEvent{Type: EventRequestShutdown, RequestId: id, Timestamp: uint64(200 + calls)}))

	require.Len(t, sink.traces, 1)
	require.Len(t, sink.traces[0].Spans, 1)
	assert.Equal(t, int64(calls), sink.traces[0].Spans[0].Calls)
	assert.Equal(t, int64(calls), sink.traces[0].Calls)
	assert.Zero(t, sink.traces[0].CallsDropped)
}
