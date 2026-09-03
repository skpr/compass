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

	switch event.Type {
	case EventRequestInit:
		return readerEncodeEvent(t, bpfRequestInitEvent{
			Type: event.Type, RequestId: event.RequestId, Method: event.Method,
			Uri: event.Uri, Timestamp: event.Timestamp,
		})
	case EventFunction:
		return readerEncodeEvent(t, bpfFunctionEvent{
			Type: event.Type, RequestId: event.RequestId, FunctionName: event.FunctionName,
			Timestamp: event.Timestamp, Elapsed: event.Elapsed, Memory: event.Memory,
		})
	case EventRequestShutdown:
		return readerEncodeEvent(t, bpfRequestShutdownEvent{
			Type: event.Type, RequestId: event.RequestId, Timestamp: event.Timestamp,
		})
	default:
		return []byte{event.Type}
	}
}

func readerEncodeEvent(t *testing.T, event any) []byte {
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

func TestCompactEventLayouts(t *testing.T) {
	assert.Equal(t, 2216, binary.Size(bpfRequestInitEvent{}))
	assert.Equal(t, 232, binary.Size(bpfFunctionEvent{}))
	assert.Equal(t, 112, binary.Size(bpfRequestShutdownEvent{}))

	const legacyPayloadSize = 2328
	functionPayloadSize := binary.Size(bpfFunctionEvent{})
	assert.GreaterOrEqual(t, legacyPayloadSize/functionPayloadSize, 8)
	assert.LessOrEqual(t, functionPayloadSize, 288)
	assert.GreaterOrEqual(t, (256*4096)/alignedRingbufRecordSize(functionPayloadSize), 3500)
}

func TestProcessEvent_RejectsMalformedRecords(t *testing.T) {
	manager, err := NewHandler(&readerTestSink{}, Options{Expire: time.Minute, Clock: clock.Monotonic{Boot: readerTestBoot}})
	require.NoError(t, err)
	skips := ingest.NewSkips(ingest.RuntimeNodeHTTP)
	valid := readerEncodeEvent(t, bpfFunctionEvent{Type: EventFunction})

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

type legacyHTTPEvent struct {
	Type         uint8
	RequestId    [101]uint8
	Method       [101]uint8
	FunctionName [101]uint8
	Uri          [2000]uint8
	Timestamp    uint64
	Elapsed      uint64
	Memory       uint64
}

var (
	benchmarkLegacyEvent  legacyHTTPEvent
	benchmarkCompactEvent bpfFunctionEvent
)

func BenchmarkDecodeFunctionEvent(b *testing.B) {
	legacyRaw := benchmarkEncodeEvent(b, legacyHTTPEvent{Type: EventFunction})
	compactRaw := benchmarkEncodeEvent(b, bpfFunctionEvent{Type: EventFunction})
	require.Equal(b, 2328, len(legacyRaw))
	require.Equal(b, 232, len(compactRaw))

	b.Run("legacy_fixed_2328_bytes", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(legacyRaw)))
		for b.Loop() {
			var err error
			benchmarkLegacyEvent, err = ingest.DecodeExact[legacyHTTPEvent](legacyRaw)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("compact_function_232_bytes", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(compactRaw)))
		for b.Loop() {
			var err error
			benchmarkCompactEvent, err = ingest.DecodeExact[bpfFunctionEvent](compactRaw)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func benchmarkEncodeEvent(b *testing.B, event any) []byte {
	b.Helper()
	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.LittleEndian, event); err != nil {
		b.Fatal(err)
	}
	return buf.Bytes()
}

func alignedRingbufRecordSize(payload int) int {
	const headerSize = 8
	return (payload + headerSize + 7) &^ 7
}

func TestCompactEvents_RoundTrip(t *testing.T) {
	requestID := readerRequestID("request-123")
	functionName := readerFunctionName("App\\Handler::run")
	var method [101]uint8
	copy(method[:], "POST")
	var uri [2000]uint8
	copy(uri[:], "/resource?full=true")

	readerAssertRoundTrip(t, bpfRequestInitEvent{
		Type: EventRequestInit, RequestId: requestID, Method: method, Uri: uri, Timestamp: 101,
	})
	readerAssertRoundTrip(t, bpfFunctionEvent{
		Type: EventFunction, RequestId: requestID, FunctionName: functionName,
		Timestamp: 202, Elapsed: 303, Memory: 404,
	})
	readerAssertRoundTrip(t, bpfRequestShutdownEvent{
		Type: EventRequestShutdown, RequestId: requestID, Timestamp: 505,
	})
}

func readerAssertRoundTrip[T comparable](t *testing.T, expected T) {
	t.Helper()
	raw := readerEncodeEvent(t, expected)
	actual, err := ingest.DecodeExact[T](raw)
	require.NoError(t, err)
	assert.Equal(t, expected, actual)
}
