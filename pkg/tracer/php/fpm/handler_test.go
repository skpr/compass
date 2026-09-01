package fpm

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skpr/compass/pkg/trace"
	"github.com/skpr/compass/pkg/tracer/clock"
)

// testBoot is the offset between the monotonic clock the probes read and the
// wall clock, fixed so the timestamps in these events land on instants the
// assertions below can name.
var testBoot = time.Unix(1700000000, 0)

// at is the instant a probe timestamp maps to.
func at(timestamp uint64) time.Time {
	return testBoot.Add(time.Duration(timestamp))
}

// mockSink captures traces sent to ProcessTrace. Guarded because requests are
// completed from either of the two ring buffer goroutines.
type mockSink struct {
	mu     sync.Mutex
	traces []trace.Trace
}

func (m *mockSink) Initialize() error { return nil }

func (m *mockSink) ProcessTrace(_ context.Context, t trace.Trace) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.traces = append(m.traces, t)

	return nil
}

// Traces captured so far.
func (m *mockSink) Traces() []trace.Trace {
	m.mu.Lock()
	defer m.mu.Unlock()

	return append([]trace.Trace(nil), m.traces...)
}

// testOptions with the clock pinned, so timestamps land on nameable instants.
func testOptions() Options {
	return Options{Expire: 5 * time.Minute, Clock: clock.Monotonic{Boot: testBoot}}
}

func newTestHandler(t *testing.T) (*Handler, *mockSink) {
	t.Helper()

	sink := &mockSink{}
	h, err := NewHandler(sink, testOptions())
	require.NoError(t, err)

	return h, sink
}

func makeRequestID(id string) [101]uint8 {
	var arr [101]uint8
	copy(arr[:], id)
	return arr
}

func makeFunctionName(name string) [101]uint8 {
	var arr [101]uint8
	copy(arr[:], name)
	return arr
}

func makeURI(uri string) [2000]uint8 {
	var arr [2000]uint8
	copy(arr[:], uri)
	return arr
}

func makeMethod(method string) [101]uint8 {
	var arr [101]uint8
	copy(arr[:], method)
	return arr
}

func TestHandler_Handle_EmptyRequestID(t *testing.T) {
	h, _ := newTestHandler(t)

	event := bpfEvent{
		Type: EventRequestInit,
		// RequestId is all zeros → empty string.
	}

	err := h.Handle(t.Context(), event)
	assert.ErrorContains(t, err, "empty request id")
}

func TestHandler_Handle_RequestInit(t *testing.T) {
	h, _ := newTestHandler(t)

	event := bpfEvent{
		Type:      EventRequestInit,
		RequestId: makeRequestID("req-1"),
		Uri:       makeURI("/api/test"),
		Method:    makeMethod("GET"),
		Timestamp: 1000,
	}

	err := h.Handle(t.Context(), event)
	require.NoError(t, err)

	// Verify the trace was stored.
	x, found := h.storage.Get("req-1")
	require.True(t, found)

	stored := x.(*state).trace
	assert.Equal(t, "req-1", stored.Metadata.ID)
	assert.Equal(t, trace.SourceHTTP, stored.Metadata.Source)
	assert.Equal(t, "/api/test", stored.Metadata.HTTP.URI)
	assert.Equal(t, "GET", stored.Metadata.HTTP.Method)
	assert.Equal(t, at(1000), stored.Metadata.StartTime)
}

func TestHandler_Handle_Function(t *testing.T) {
	h, _ := newTestHandler(t)

	// First, init the request.
	initEvent := bpfEvent{
		Type:      EventRequestInit,
		RequestId: makeRequestID("req-1"),
		Uri:       makeURI("/test"),
		Method:    makeMethod("GET"),
		Timestamp: 1000,
	}
	require.NoError(t, h.Handle(t.Context(), initEvent))

	// Then, send a function event.
	funcEvent := bpfEvent{
		Type:         EventFunction,
		RequestId:    makeRequestID("req-1"),
		FunctionName: makeFunctionName("myFunc"),
		Timestamp:    1500,
		Elapsed:      200,
		Memory:       4096,
	}
	require.NoError(t, h.Handle(t.Context(), funcEvent))

	// Verify the function was stored.
	x, found := h.storage.Get("req-1")
	require.True(t, found)

	stored := x.(*state).trace
	require.Len(t, stored.FunctionCalls, 1)
	assert.Equal(t, "myFunc", stored.FunctionCalls[0].Name)
	assert.Equal(t, 300*time.Nanosecond, stored.FunctionCalls[0].Offset) // (1500 - 200) into a request which began at 1000
	assert.Equal(t, 200*time.Nanosecond, stored.FunctionCalls[0].Elapsed)
	assert.Equal(t, int64(4096), stored.FunctionCalls[0].Memory)
	assert.Equal(t, int64(4096), stored.ResourceUtilisation.MaxMemory)
}

func TestHandler_Handle_Function_NotFound(t *testing.T) {
	h, _ := newTestHandler(t)

	event := bpfEvent{
		Type:         EventFunction,
		RequestId:    makeRequestID("req-nonexistent"),
		FunctionName: makeFunctionName("myFunc"),
		Timestamp:    1500,
		Elapsed:      200,
		Memory:       4096,
	}

	err := h.Handle(t.Context(), event)
	assert.ErrorContains(t, err, "not found in storage")
}

func TestHandler_Handle_RequestShutdown(t *testing.T) {
	h, sink := newTestHandler(t)

	// Init.
	require.NoError(t, h.Handle(t.Context(), bpfEvent{
		Type:      EventRequestInit,
		RequestId: makeRequestID("req-1"),
		Uri:       makeURI("/test"),
		Method:    makeMethod("POST"),
		Timestamp: 1000,
	}))

	// Function.
	require.NoError(t, h.Handle(t.Context(), bpfEvent{
		Type:         EventFunction,
		RequestId:    makeRequestID("req-1"),
		FunctionName: makeFunctionName("handler"),
		Timestamp:    1500,
		Elapsed:      300,
		Memory:       2048,
	}))

	// Shutdown.
	err := h.Handle(t.Context(), bpfEvent{
		Type:      EventRequestShutdown,
		RequestId: makeRequestID("req-1"),
		Timestamp: 2000,
	})
	require.NoError(t, err)

	// Verify trace was sent to sink.
	require.Len(t, sink.traces, 1)
	assert.Equal(t, "req-1", sink.traces[0].Metadata.ID)
	assert.Equal(t, at(2000), sink.traces[0].Metadata.EndTime)
	assert.Len(t, sink.traces[0].FunctionCalls, 1)

	// Verify storage was cleaned up.
	_, found := h.storage.Get("req-1")
	assert.False(t, found)
}

func TestHandler_Handle_RequestShutdown_NotFound(t *testing.T) {
	h, _ := newTestHandler(t)

	err := h.Handle(t.Context(), bpfEvent{
		Type:      EventRequestShutdown,
		RequestId: makeRequestID("req-nonexistent"),
		Timestamp: 2000,
	})
	assert.ErrorContains(t, err, "not found in storage")
}

func TestHandler_Handle_RequestShutdown_NoFunctions(t *testing.T) {
	h, _ := newTestHandler(t)

	// Init without any functions.
	require.NoError(t, h.Handle(t.Context(), bpfEvent{
		Type:      EventRequestInit,
		RequestId: makeRequestID("req-1"),
		Uri:       makeURI("/test"),
		Method:    makeMethod("GET"),
		Timestamp: 1000,
	}))

	// Shutdown immediately.
	err := h.Handle(t.Context(), bpfEvent{
		Type:      EventRequestShutdown,
		RequestId: makeRequestID("req-1"),
		Timestamp: 2000,
	})
	assert.ErrorContains(t, err, "no functions found")
}

func TestHandler_Handle_FullLifecycle(t *testing.T) {
	h, sink := newTestHandler(t)

	// Init.
	require.NoError(t, h.Handle(t.Context(), bpfEvent{
		Type:      EventRequestInit,
		RequestId: makeRequestID("req-full"),
		Uri:       makeURI("/lifecycle"),
		Method:    makeMethod("GET"),
		Timestamp: 1000,
	}))

	// Multiple functions.
	require.NoError(t, h.Handle(t.Context(), bpfEvent{
		Type:         EventFunction,
		RequestId:    makeRequestID("req-full"),
		FunctionName: makeFunctionName("funcA"),
		Timestamp:    1500,
		Elapsed:      200,
		Memory:       1024,
	}))
	require.NoError(t, h.Handle(t.Context(), bpfEvent{
		Type:         EventFunction,
		RequestId:    makeRequestID("req-full"),
		FunctionName: makeFunctionName("funcB"),
		Timestamp:    2000,
		Elapsed:      300,
		Memory:       8192,
	}))

	// Shutdown.
	require.NoError(t, h.Handle(t.Context(), bpfEvent{
		Type:      EventRequestShutdown,
		RequestId: makeRequestID("req-full"),
		Timestamp: 3000,
	}))

	// Verify complete trace.
	require.Len(t, sink.traces, 1)

	tr := sink.traces[0]
	assert.Equal(t, "req-full", tr.Metadata.ID)
	assert.Equal(t, trace.SourceHTTP, tr.Metadata.Source)
	assert.Equal(t, "GET", tr.Metadata.HTTP.Method)
	assert.Equal(t, "/lifecycle", tr.Metadata.HTTP.URI)
	assert.Equal(t, at(1000), tr.Metadata.StartTime)
	assert.Equal(t, at(3000), tr.Metadata.EndTime)
	assert.Len(t, tr.FunctionCalls, 2)
	assert.Equal(t, int64(8192), tr.ResourceUtilisation.MaxMemory)
}
