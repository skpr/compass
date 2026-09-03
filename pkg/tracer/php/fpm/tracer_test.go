package fpm

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skpr/compass/pkg/trace"
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
	skips := ingest.NewSkips(ingest.RuntimePHPFPM)
	events := []bpfEvent{
		{Type: EventFunction, RequestId: makeRequestID("orphan-function")},
		{Type: EventRequestShutdown, RequestId: makeRequestID("orphan-shutdown")},
		{Type: EventRequestInit},
		{Type: EventRequestInit, RequestId: makeRequestID("empty-trace"), Timestamp: 10},
		{Type: EventRequestShutdown, RequestId: makeRequestID("empty-trace"), Timestamp: 20},
		{Type: EventRequestInit, RequestId: makeRequestID("complete"), Timestamp: 100},
		{Type: EventFunction, RequestId: makeRequestID("complete"), FunctionName: makeFunctionName("handler"), Timestamp: 150, Elapsed: 25},
		{Type: EventRequestShutdown, RequestId: makeRequestID("complete"), Timestamp: 200},
	}

	for _, event := range events {
		require.NoError(t, processEvent(t.Context(), fpmRawEvent(t, event), manager, skips))
	}

	assert.Equal(t, uint64(2), skips.Count(ingest.ReasonRequestNotTracked))
	assert.Equal(t, uint64(1), skips.Count(ingest.ReasonInvalidIdentifier))
	assert.Equal(t, uint64(1), skips.Count(ingest.ReasonTraceEmpty))
	traces := sink.Traces()
	require.Len(t, traces, 1)
	assert.Equal(t, "complete", traces[0].Metadata.ID)
}

func TestProcessEvent_FatalErrors(t *testing.T) {
	t.Run("decode", func(t *testing.T) {
		manager, _ := newTestHandler(t)
		err := processEvent(t.Context(), []byte{1}, manager, ingest.NewSkips(ingest.RuntimePHPFPM))
		assert.ErrorContains(t, err, "failed to read event")
	})

	t.Run("sink", func(t *testing.T) {
		sinkErr := errors.New("sink failed")
		manager, err := NewHandler(&readerErrorSink{err: sinkErr}, testOptions())
		require.NoError(t, err)
		skips := ingest.NewSkips(ingest.RuntimePHPFPM)

		for _, event := range []bpfEvent{
			{Type: EventRequestInit, RequestId: makeRequestID("fatal"), Timestamp: 100},
			{Type: EventFunction, RequestId: makeRequestID("fatal"), Timestamp: 150, Elapsed: 25},
		} {
			require.NoError(t, processEvent(t.Context(), fpmRawEvent(t, event), manager, skips))
		}

		err = processEvent(t.Context(), fpmRawEvent(t, bpfEvent{Type: EventRequestShutdown, RequestId: makeRequestID("fatal"), Timestamp: 200}), manager, skips)
		assert.ErrorIs(t, err, sinkErr)
		assert.Zero(t, skips.Total())
	})
}

func TestProcessDrupalCacheEvent_OnlySkipsTypedOutcomes(t *testing.T) {
	manager, _ := newTestHandler(t)
	skips := ingest.NewSkips(ingest.RuntimePHPFPM)

	orphan := bpfDrupalCacheEvent{Type: EventDrupalCacheRenderArray, RequestId: makeRequestID("orphan")}
	require.NoError(t, processDrupalCacheEvent(t.Context(), fpmRawDrupalEvent(t, orphan), manager, skips))
	assert.Equal(t, uint64(1), skips.Count(ingest.ReasonRequestNotTracked))

	unknown := bpfDrupalCacheEvent{Type: 255, RequestId: makeRequestID("malformed")}
	err := processDrupalCacheEvent(t.Context(), fpmRawDrupalEvent(t, unknown), manager, skips)
	assert.ErrorContains(t, err, "unknown drupal cache event type")
}

func fpmRawEvent(t *testing.T, event bpfEvent) []byte {
	t.Helper()

	switch event.Type {
	case EventRequestInit:
		return fpmEncodeEvent(t, bpfRequestInitEvent{
			Type: event.Type, RequestId: event.RequestId, Method: event.Method,
			Uri: event.Uri, Timestamp: event.Timestamp,
		})
	case EventFunction:
		return fpmEncodeEvent(t, bpfFunctionEvent{
			Type: event.Type, RequestId: event.RequestId, FunctionName: event.FunctionName,
			Timestamp: event.Timestamp, Elapsed: event.Elapsed, Memory: event.Memory,
		})
	case EventRequestShutdown:
		return fpmEncodeEvent(t, bpfRequestShutdownEvent{
			Type: event.Type, RequestId: event.RequestId, Timestamp: event.Timestamp,
		})
	default:
		return []byte{event.Type}
	}
}

func fpmEncodeEvent(t *testing.T, event any) []byte {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, binary.Write(&buf, binary.LittleEndian, event))
	return buf.Bytes()
}

func fpmRawDrupalEvent(t *testing.T, event bpfDrupalCacheEvent) []byte {
	t.Helper()

	var buf bytes.Buffer
	require.NoError(t, binary.Write(&buf, binary.LittleEndian, event))
	return buf.Bytes()
}

func TestCompactEventLayouts(t *testing.T) {
	assert.Equal(t, 2216, binary.Size(bpfRequestInitEvent{}))
	assert.Equal(t, 232, binary.Size(bpfFunctionEvent{}))
	assert.Equal(t, 112, binary.Size(bpfRequestShutdownEvent{}))

	const legacyPayloadSize = 2328
	functionPayloadSize := binary.Size(bpfFunctionEvent{})
	assert.GreaterOrEqual(t, legacyPayloadSize/functionPayloadSize, 8)
	assert.LessOrEqual(t, functionPayloadSize, 288)
	assert.GreaterOrEqual(t, (256*4096)/fpmAlignedRingbufRecordSize(functionPayloadSize), 3500)
}

func TestProcessEvent_RejectsMalformedRecords(t *testing.T) {
	manager, _ := newTestHandler(t)
	skips := ingest.NewSkips(ingest.RuntimePHPFPM)
	valid := fpmEncodeEvent(t, bpfFunctionEvent{Type: EventFunction})

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

func fpmAlignedRingbufRecordSize(payload int) int {
	const headerSize = 8
	return (payload + headerSize + 7) &^ 7
}

func TestCompactEvents_RoundTrip(t *testing.T) {
	requestID := makeRequestID("request-123")

	fpmAssertRoundTrip(t, bpfRequestInitEvent{
		Type: EventRequestInit, RequestId: requestID, Method: makeMethod("POST"),
		Uri: makeURI("/resource?full=true"), Timestamp: 101,
	})
	fpmAssertRoundTrip(t, bpfFunctionEvent{
		Type: EventFunction, RequestId: requestID, FunctionName: makeFunctionName("App\\Handler::run"),
		Timestamp: 202, Elapsed: 303, Memory: 404,
	})
	fpmAssertRoundTrip(t, bpfRequestShutdownEvent{
		Type: EventRequestShutdown, RequestId: requestID, Timestamp: 505,
	})
}

func fpmAssertRoundTrip[T comparable](t *testing.T, expected T) {
	t.Helper()
	raw := fpmEncodeEvent(t, expected)
	actual, err := ingest.DecodeExact[T](raw)
	require.NoError(t, err)
	assert.Equal(t, expected, actual)
}
