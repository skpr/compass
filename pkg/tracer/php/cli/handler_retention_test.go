package cli

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skpr/compass/pkg/trace"
	"github.com/skpr/compass/pkg/tracer/clock"
)

func TestHandler_FunctionCallsAreBounded(t *testing.T) {
	sink := &mockSink{}
	h, err := NewHandler(sink, Options{
		Expire:           time.Minute,
		MaxFunctionCalls: 2,
		Clock:            clock.Monotonic{Boot: testBoot},
	})
	require.NoError(t, err)

	const pid = 42
	require.NoError(t, h.Handle(t.Context(), bpfEvent{Type: EventRequestInit, Pid: pid, Timestamp: 100}))

	for i, memory := range []uint64{10, 20, 30, 40} {
		require.NoError(t, h.Handle(t.Context(), bpfEvent{
			Type: EventFunction, Pid: pid,
			FunctionName: makeFunctionName("function"),
			Timestamp:    uint64(200 + i), Elapsed: 1, Memory: memory,
		}))
	}

	stored, found := h.storage.Get("42")
	require.True(t, found)
	tr := stored.(trace.Trace)
	assert.Len(t, tr.FunctionCalls, 2)
	assert.Equal(t, 2, tr.FunctionCallsDropped)
	assert.Equal(t, int64(40), tr.ResourceUtilisation.MaxMemory)

	require.NoError(t, h.Handle(t.Context(), bpfEvent{Type: EventRequestShutdown, Pid: pid, Timestamp: 300}))
	require.Len(t, sink.traces, 1)
	assert.Equal(t, 2, sink.traces[0].FunctionCallsDropped)
}
