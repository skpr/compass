package cli

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skpr/compass/pkg/trace"
	"github.com/skpr/compass/pkg/tracer/clock"
	"github.com/skpr/compass/pkg/tracer/ingest"
)

type readerErrorSink struct {
	err error
}

func (s *readerErrorSink) Initialize() error { return nil }

func (s *readerErrorSink) ProcessTrace(_ context.Context, _ trace.Trace) error {
	return s.err
}

func TestProcessEvent_ContinuesAfterSkippedEvents(t *testing.T) {
	manager, sink := newTestHandler(t)
	skips := ingest.NewSkips(ingest.RuntimePHPCLI)
	events := []bpfEvent{
		{Type: EventFunction, Pid: 10},
		{Type: EventRequestShutdown, Pid: 11},
		{Type: EventRequestInit},
		{Type: EventRequestInit, Pid: 12, Timestamp: 10},
		{Type: EventRequestShutdown, Pid: 12, Timestamp: 20},
		{Type: EventRequestInit, Pid: 13, Command: makeCommand("php complete.php"), Timestamp: 100},
		{Type: EventFunction, Pid: 13, FunctionName: makeFunctionName("handler"), Timestamp: 150, Elapsed: 25},
		{Type: EventRequestShutdown, Pid: 13, Timestamp: 200},
	}

	for _, event := range events {
		require.NoError(t, processEvent(t.Context(), cliRawEvent(t, event), manager, skips))
	}

	assert.Equal(t, uint64(2), skips.Count(ingest.ReasonRequestNotTracked))
	assert.Equal(t, uint64(1), skips.Count(ingest.ReasonInvalidIdentifier))
	assert.Equal(t, uint64(1), skips.Count(ingest.ReasonTraceEmpty))
	require.Len(t, sink.traces, 1)
	assert.Equal(t, "13", sink.traces[0].Metadata.ID)
}

func TestProcessEvent_FatalErrors(t *testing.T) {
	t.Run("decode", func(t *testing.T) {
		manager, _ := newTestHandler(t)
		err := processEvent(t.Context(), []byte{1}, manager, ingest.NewSkips(ingest.RuntimePHPCLI))
		assert.ErrorContains(t, err, "failed to read event")
	})

	t.Run("sink", func(t *testing.T) {
		sinkErr := errors.New("sink failed")
		manager, err := NewHandler(&readerErrorSink{err: sinkErr}, Options{
			Expire: time.Minute,
			Clock:  clock.Monotonic{Boot: testBoot},
		})
		require.NoError(t, err)
		skips := ingest.NewSkips(ingest.RuntimePHPCLI)

		for _, event := range []bpfEvent{
			{Type: EventRequestInit, Pid: 13, Timestamp: 100},
			{Type: EventFunction, Pid: 13, Timestamp: 150, Elapsed: 25},
		} {
			require.NoError(t, processEvent(t.Context(), cliRawEvent(t, event), manager, skips))
		}

		err = processEvent(t.Context(), cliRawEvent(t, bpfEvent{Type: EventRequestShutdown, Pid: 13, Timestamp: 200}), manager, skips)
		assert.ErrorIs(t, err, sinkErr)
		assert.Zero(t, skips.Total())
	})
}

func cliRawEvent(t *testing.T, event bpfEvent) []byte {
	t.Helper()

	switch event.Type {
	case EventRequestInit:
		return cliEncodeEvent(t, bpfRequestInitEvent{
			Type: event.Type, Command: event.Command, Pid: event.Pid, Timestamp: event.Timestamp,
		})
	case EventFunction:
		return cliEncodeEvent(t, bpfFunctionEvent{
			Type: event.Type, FunctionName: event.FunctionName, Pid: event.Pid,
			Timestamp: event.Timestamp, Elapsed: event.Elapsed, Memory: event.Memory,
		})
	case EventRequestShutdown:
		return cliEncodeEvent(t, bpfRequestShutdownEvent{
			Type: event.Type, Pid: event.Pid, Timestamp: event.Timestamp,
		})
	default:
		return []byte{event.Type}
	}
}

func cliEncodeEvent(t *testing.T, event any) []byte {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, binary.Write(&buf, binary.LittleEndian, event))
	return buf.Bytes()
}

func TestCompactEventLayouts(t *testing.T) {
	assert.Equal(t, 120, binary.Size(bpfRequestInitEvent{}))
	assert.Equal(t, 136, binary.Size(bpfFunctionEvent{}))
	assert.Equal(t, 24, binary.Size(bpfRequestShutdownEvent{}))
}

func TestProcessEvent_RejectsMalformedRecords(t *testing.T) {
	manager, _ := newTestHandler(t)
	skips := ingest.NewSkips(ingest.RuntimePHPCLI)
	valid := cliEncodeEvent(t, bpfFunctionEvent{Type: EventFunction})

	tests := map[string][]byte{
		"empty":     {},
		"truncated": valid[:len(valid)-1],
		"oversized": append(append([]byte{}, valid...), 0),
		"unknown":   {255},
	}
	for name, raw := range tests {
		t.Run(name, func(t *testing.T) {
			err := processEvent(t.Context(), raw, manager, skips)
			assert.ErrorContains(t, err, "failed to read event")
		})
	}
}

func TestCompactEvents_RoundTrip(t *testing.T) {
	const pid = uint64(1234)

	cliAssertRoundTrip(t, bpfRequestInitEvent{
		Type: EventRequestInit, Command: makeCommand("php script.php --full"), Pid: pid, Timestamp: 101,
	})
	cliAssertRoundTrip(t, bpfFunctionEvent{
		Type: EventFunction, FunctionName: makeFunctionName("App\\Command::run"), Pid: pid,
		Timestamp: 202, Elapsed: 303, Memory: 404,
	})
	cliAssertRoundTrip(t, bpfRequestShutdownEvent{
		Type: EventRequestShutdown, Pid: pid, Timestamp: 505,
	})
}

func cliAssertRoundTrip[T comparable](t *testing.T, expected T) {
	t.Helper()
	raw := cliEncodeEvent(t, expected)
	actual, err := ingest.DecodeExact[T](raw)
	require.NoError(t, err)
	assert.Equal(t, expected, actual)
}
