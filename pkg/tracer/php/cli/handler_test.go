package cli

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skpr/compass/pkg/trace"
)

// mockSink captures traces sent to ProcessTrace.
type mockSink struct {
	traces []trace.Trace
}

func (m *mockSink) Initialize() error { return nil }

func (m *mockSink) ProcessTrace(_ context.Context, t trace.Trace) error {
	m.traces = append(m.traces, t)
	return nil
}

func newTestHandler(t *testing.T) (*Handler, *mockSink) {
	t.Helper()

	sink := &mockSink{}
	h, err := NewHandler(sink, Options{Expire: 5 * time.Minute})
	require.NoError(t, err)

	return h, sink
}

func makeCommand(cmd string) [101]uint8 {
	var arr [101]uint8
	copy(arr[:], cmd)
	return arr
}

func makeFunctionName(name string) [101]uint8 {
	var arr [101]uint8
	copy(arr[:], name)
	return arr
}

func TestHandler_Handle_EmptyPID(t *testing.T) {
	h, _ := newTestHandler(t)

	event := bpfEvent{
		Type: EventRequestInit,
		Pid:  0,
	}

	err := h.Handle(event)
	assert.ErrorContains(t, err, "empty pid")
}

func TestHandler_Handle_RequestInit(t *testing.T) {
	h, _ := newTestHandler(t)

	event := bpfEvent{
		Type:      EventRequestInit,
		Pid:       42,
		Command:   makeCommand("drush cr"),
		Timestamp: 1000,
	}

	err := h.Handle(event)
	require.NoError(t, err)

	// Verify the trace was stored.
	x, found := h.storage.Get("42")
	require.True(t, found)

	stored := x.(trace.Trace)
	assert.Equal(t, "42", stored.Metadata.ID)
	assert.Equal(t, trace.SourceCLI, stored.Metadata.Source)
	assert.Equal(t, "drush cr", stored.Metadata.CLI.Command)
	assert.Equal(t, int64(1000), stored.Metadata.StartTime)
}

func TestHandler_Handle_Function(t *testing.T) {
	h, _ := newTestHandler(t)

	// Init.
	require.NoError(t, h.Handle(bpfEvent{
		Type:      EventRequestInit,
		Pid:       42,
		Command:   makeCommand("drush cr"),
		Timestamp: 1000,
	}))

	// Function.
	err := h.Handle(bpfEvent{
		Type:         EventFunction,
		Pid:          42,
		FunctionName: makeFunctionName("myFunc"),
		Timestamp:    1500,
		Elapsed:      200,
		Memory:       4096,
	})
	require.NoError(t, err)

	x, found := h.storage.Get("42")
	require.True(t, found)

	stored := x.(trace.Trace)
	require.Len(t, stored.FunctionCalls, 1)
	assert.Equal(t, "myFunc", stored.FunctionCalls[0].Name)
	assert.Equal(t, int64(1300), stored.FunctionCalls[0].StartTime) // 1500 - 200
	assert.Equal(t, int64(200), stored.FunctionCalls[0].Elapsed)
	assert.Equal(t, int64(4096), stored.ResourceUtilisation.MaxMemory)
}

func TestHandler_Handle_Function_NotFound(t *testing.T) {
	h, _ := newTestHandler(t)

	err := h.Handle(bpfEvent{
		Type:         EventFunction,
		Pid:          999,
		FunctionName: makeFunctionName("myFunc"),
		Timestamp:    1500,
		Elapsed:      200,
		Memory:       4096,
	})
	assert.ErrorContains(t, err, "not found in storage")
}

func TestHandler_Handle_RequestShutdown(t *testing.T) {
	h, sink := newTestHandler(t)

	// Init.
	require.NoError(t, h.Handle(bpfEvent{
		Type:      EventRequestInit,
		Pid:       42,
		Command:   makeCommand("drush status"),
		Timestamp: 1000,
	}))

	// Function.
	require.NoError(t, h.Handle(bpfEvent{
		Type:         EventFunction,
		Pid:          42,
		FunctionName: makeFunctionName("handler"),
		Timestamp:    1500,
		Elapsed:      300,
		Memory:       2048,
	}))

	// Shutdown.
	err := h.Handle(bpfEvent{
		Type:      EventRequestShutdown,
		Pid:       42,
		Timestamp: 2000,
	})
	require.NoError(t, err)

	require.Len(t, sink.traces, 1)
	assert.Equal(t, "42", sink.traces[0].Metadata.ID)
	assert.Equal(t, int64(2000), sink.traces[0].Metadata.EndTime)
	assert.Len(t, sink.traces[0].FunctionCalls, 1)

	// Verify storage was cleaned up.
	_, found := h.storage.Get("42")
	assert.False(t, found)
}

func TestHandler_Handle_FullLifecycle(t *testing.T) {
	h, sink := newTestHandler(t)

	// Init.
	require.NoError(t, h.Handle(bpfEvent{
		Type:      EventRequestInit,
		Pid:       100,
		Command:   makeCommand("php script.php"),
		Timestamp: 1000,
	}))

	// Multiple functions.
	require.NoError(t, h.Handle(bpfEvent{
		Type:         EventFunction,
		Pid:          100,
		FunctionName: makeFunctionName("funcA"),
		Timestamp:    1500,
		Elapsed:      200,
		Memory:       1024,
	}))
	require.NoError(t, h.Handle(bpfEvent{
		Type:         EventFunction,
		Pid:          100,
		FunctionName: makeFunctionName("funcB"),
		Timestamp:    2000,
		Elapsed:      300,
		Memory:       8192,
	}))

	// Shutdown.
	require.NoError(t, h.Handle(bpfEvent{
		Type:      EventRequestShutdown,
		Pid:       100,
		Timestamp: 3000,
	}))

	require.Len(t, sink.traces, 1)

	tr := sink.traces[0]
	assert.Equal(t, "100", tr.Metadata.ID)
	assert.Equal(t, trace.SourceCLI, tr.Metadata.Source)
	assert.Equal(t, "php script.php", tr.Metadata.CLI.Command)
	assert.Equal(t, int64(1000), tr.Metadata.StartTime)
	assert.Equal(t, int64(3000), tr.Metadata.EndTime)
	assert.Len(t, tr.FunctionCalls, 2)
	assert.Equal(t, int64(8192), tr.ResourceUtilisation.MaxMemory)
}
