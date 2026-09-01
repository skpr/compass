package http

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

var readerTestBoot = time.Unix(1700000000, 0)

type readerTestSink struct {
	traces []trace.Trace
	err    error
}

func (s *readerTestSink) Initialize() error { return nil }

func (s *readerTestSink) ProcessTrace(_ context.Context, tr trace.Trace) error {
	if s.err != nil {
		return s.err
	}

	s.traces = append(s.traces, tr)
	return nil
}

func TestProcessEvent_ContinuesAfterSkippedEvents(t *testing.T) {
	sink := &readerTestSink{}
	manager, err := NewHandler(sink, Options{
		Expire: time.Minute,
		Clock:  clock.Monotonic{Boot: readerTestBoot},
	})
	require.NoError(t, err)

	skips := ingest.NewSkips(ingest.RuntimeNodeHTTP)
	events := []bpfEvent{
		{Type: EventFunction, RequestId: readerRequestID("orphan-function")},
		{Type: EventRequestShutdown, RequestId: readerRequestID("orphan-shutdown")},
		{Type: EventRequestInit},
		{Type: EventRequestInit, RequestId: readerRequestID("empty-trace"), Timestamp: 10},
		{Type: EventRequestShutdown, RequestId: readerRequestID("empty-trace"), Timestamp: 20},
		{Type: EventRequestInit, RequestId: readerRequestID("complete"), Timestamp: 100},
		{Type: EventFunction, RequestId: readerRequestID("complete"), FunctionName: readerFunctionName("handler"), Timestamp: 150, Elapsed: 25},
		{Type: EventRequestShutdown, RequestId: readerRequestID("complete"), Timestamp: 200},
	}

	for _, event := range events {
		require.NoError(t, processEvent(t.Context(), readerRawEvent(t, event), manager, skips))
	}

	assert.Equal(t, uint64(2), skips.Count(ingest.ReasonRequestNotTracked))
	assert.Equal(t, uint64(1), skips.Count(ingest.ReasonInvalidIdentifier))
	assert.Equal(t, uint64(1), skips.Count(ingest.ReasonTraceEmpty))
	require.Len(t, sink.traces, 1)
	assert.Equal(t, "complete", sink.traces[0].Metadata.ID)
}

func TestProcessEvent_FatalErrors(t *testing.T) {
	t.Run("decode", func(t *testing.T) {
		manager, err := NewHandler(&readerTestSink{}, Options{Expire: time.Minute, Clock: clock.Monotonic{Boot: readerTestBoot}})
		require.NoError(t, err)

		err = processEvent(t.Context(), []byte{1}, manager, ingest.NewSkips(ingest.RuntimeNodeHTTP))
		assert.ErrorContains(t, err, "failed to read event")
	})

	t.Run("sink", func(t *testing.T) {
		sinkErr := errors.New("sink failed")
		manager, err := NewHandler(&readerTestSink{err: sinkErr}, Options{Expire: time.Minute, Clock: clock.Monotonic{Boot: readerTestBoot}})
		require.NoError(t, err)
		skips := ingest.NewSkips(ingest.RuntimeNodeHTTP)

		for _, event := range []bpfEvent{
			{Type: EventRequestInit, RequestId: readerRequestID("fatal"), Timestamp: 100},
			{Type: EventFunction, RequestId: readerRequestID("fatal"), Timestamp: 150, Elapsed: 25},
		} {
			require.NoError(t, processEvent(t.Context(), readerRawEvent(t, event), manager, skips))
		}

		err = processEvent(t.Context(), readerRawEvent(t, bpfEvent{Type: EventRequestShutdown, RequestId: readerRequestID("fatal"), Timestamp: 200}), manager, skips)
		assert.ErrorIs(t, err, sinkErr)
		assert.Zero(t, skips.Total())
	})
}

func readerRawEvent(t *testing.T, event bpfEvent) []byte {
	t.Helper()

	var buf bytes.Buffer
	require.NoError(t, binary.Write(&buf, binary.LittleEndian, event))
	return buf.Bytes()
}

func readerRequestID(id string) [101]uint8 {
	var value [101]uint8
	copy(value[:], id)
	return value
}

func readerFunctionName(name string) [101]uint8 {
	var value [101]uint8
	copy(value[:], name)
	return value
}
