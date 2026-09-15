package http

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skpr/compass/pkg/tracer/clock"
)

// noisyRequestID is the id field as a probe really fills it: the string and
// its terminator written into a record the ring buffer handed over dirty, so
// everything past the terminator is whatever the last record left there.
func noisyRequestID(id string, noise uint8) [101]uint8 {
	var field [101]uint8

	for i := range field {
		field[i] = noise
	}

	copy(field[:], id)
	field[len(id)] = 0

	return field
}

// Two events of the same request carry the same id and different rubbish
// behind it, so what they are matched by has to be what the field says rather
// than what it holds. Keying on the field as it stands loses every function
// event, and with it every trace.
func TestHandler_MatchesARequestAcrossDirtyRecords(t *testing.T) {
	sink := &readerTestSink{}
	h, err := NewHandler(sink, Options{
		Expire: time.Minute,
		Clock:  clock.Monotonic{Boot: readerTestBoot},
	})
	require.NoError(t, err)

	require.NoError(t, h.Handle(t.Context(), bpfEvent{
		Type:      EventRequestInit,
		RequestId: noisyRequestID("req-1", 0x00),
		Timestamp: 1000,
	}))

	require.NoError(t, h.Handle(t.Context(), bpfEvent{
		Type:         EventFunction,
		RequestId:    noisyRequestID("req-1", 0xAB),
		FunctionName: readerFunctionName("handler"),
		Timestamp:    1500,
		Elapsed:      200,
		Memory:       4096,
	}))

	require.NoError(t, h.Handle(t.Context(), bpfEvent{
		Type:      EventRequestShutdown,
		RequestId: noisyRequestID("req-1", 0xCD),
		Timestamp: 2000,
	}))

	require.Len(t, sink.traces, 1, "the request was not matched across its events")
	assert.Equal(t, "req-1", sink.traces[0].Metadata.ID)
	require.Len(t, sink.traces[0].Spans, 1)
	assert.Equal(t, "handler", sink.traces[0].Spans[0].Name)
}
