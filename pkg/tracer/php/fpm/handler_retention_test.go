package fpm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandler_FunctionCallsAreBounded(t *testing.T) {
	sink := &mockSink{}
	options := testOptions()
	options.MaxFunctionCalls = 2
	h, err := NewHandler(sink, options)
	require.NoError(t, err)

	id := makeRequestID("bounded")
	require.NoError(t, h.Handle(t.Context(), bpfEvent{Type: EventRequestInit, RequestId: id, Timestamp: 100}))

	for i, memory := range []uint64{10, 20, 30, 40} {
		require.NoError(t, h.Handle(t.Context(), bpfEvent{
			Type: EventFunction, RequestId: id,
			FunctionName: makeFunctionName("function"),
			Timestamp:    uint64(200 + i), Elapsed: 1, Memory: memory,
		}))
	}

	stored, found := h.storage.Get("bounded")
	require.True(t, found)
	tr := stored.(*state).trace
	assert.Len(t, tr.FunctionCalls, 2)
	assert.Equal(t, 2, tr.FunctionCallsDropped)
	assert.Equal(t, int64(40), tr.ResourceUtilisation.MaxMemory)

	require.NoError(t, h.Handle(t.Context(), bpfEvent{Type: EventRequestShutdown, RequestId: id, Timestamp: 300}))
	traces := sink.Traces()
	require.Len(t, traces, 1)
	assert.Equal(t, 2, traces[0].FunctionCallsDropped)
}

func TestHandler_FunctionCallsStayBoundedAtOneHundredThousandEvents(t *testing.T) {
	const retained = 64

	options := testOptions()
	options.MaxFunctionCalls = retained
	h, err := NewHandler(&mockSink{}, options)
	require.NoError(t, err)

	id := makeRequestID("stress")
	require.NoError(t, h.Handle(t.Context(), bpfEvent{Type: EventRequestInit, RequestId: id, Timestamp: 100}))

	for i := 0; i < 100_000; i++ {
		require.NoError(t, h.Handle(t.Context(), bpfEvent{
			Type: EventFunction, RequestId: id,
			FunctionName: makeFunctionName("hot_path"),
			Timestamp:    uint64(200 + i), Elapsed: 1, Memory: uint64(i),
		}))
	}

	stored, found := h.storage.Get("stress")
	require.True(t, found)
	tr := stored.(*state).trace
	assert.Len(t, tr.FunctionCalls, retained)
	assert.Equal(t, 100_000-retained, tr.FunctionCallsDropped)
	assert.Equal(t, int64(99_999), tr.ResourceUtilisation.MaxMemory)
	assert.LessOrEqual(t, cap(tr.FunctionCalls), retained*2)
}
