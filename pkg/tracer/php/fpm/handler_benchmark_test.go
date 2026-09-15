package fpm

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/skpr/compass/pkg/tracer/clock"
	"github.com/skpr/compass/pkg/tracer/spans"
)

// BenchmarkHandleFunction is what one function call costs the handler: the
// lookup which keeps its request alive, and retaining the call.
//
// This is the per-event cost which, with the decode in front of it, bounds how
// fast a collector drains its ring buffer. A Drupal request can produce more
// than a million of these, so the whole budget for one of them is nanoseconds.
func BenchmarkHandleFunction(b *testing.B) {
	// Calls per request, so that the benchmark measures the steady state.
	// Letting one request run for the whole benchmark would measure the
	// garbage collector walking an aggregate no request ever produces.
	const callsPerRequest = 10_000

	handler, err := NewHandler(&mockSink{}, Options{
		Expire: time.Minute,
		Spans:  spans.Options{Max: callsPerRequest},
		Clock:  clock.Monotonic{Boot: time.Unix(1700000000, 0)},
	})
	require.NoError(b, err)

	requestID := makeRequestID("0123456789abcdef0123456789abcdef")

	init := bpfRequestInitEvent{
		Type:      EventRequestInit,
		RequestId: requestID,
		Method:    makeMethod("GET"),
		Timestamp: 1000,
	}

	event := bpfFunctionEvent{
		Type:         EventFunction,
		RequestId:    requestID,
		FunctionName: makeFunctionName("Drupal\\Core\\Entity\\Sql\\SqlContentEntityStorage::loadMultiple"),
		Timestamp:    2000,
		Elapsed:      500,
		Memory:       1 << 20,
	}

	ctx := b.Context()

	b.ReportAllocs()

	for calls := 0; b.Loop(); calls++ {
		if calls%callsPerRequest == 0 {
			init.Timestamp = event.Timestamp

			if err := handler.HandleRequestInit(ctx, init); err != nil {
				b.Fatal(err)
			}
		}

		event.Timestamp += 1000

		if err := handler.HandleFunction(ctx, event); err != nil {
			b.Fatal(err)
		}
	}
}
