package fpm

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skpr/compass/pkg/trace"
)

// The tracer reads the request lifecycle and the Drupal cacheability from two
// ring buffers, in two goroutines, with nothing ordering one against the other.

// functionEvents sends n function calls for a request.
func functionEvents(ctx context.Context, h *Handler, requestID string, n int) {
	for i := range n {
		_ = h.Handle(ctx, bpfEvent{
			Type:         EventFunction,
			RequestId:    makeRequestID(requestID),
			FunctionName: makeFunctionName(fmt.Sprintf("Drupal\\Core\\Render\\Renderer::render%d", i)),
			Timestamp:    uint64(2000 + i*10),
			Elapsed:      5,
			Memory:       uint64(1024 * i),
		})
	}
}

// cacheEvents sends n distinct Drupal cache events for a request, from index
// first. Distinct because identical ones aggregate into one entry.
func cacheEvents(ctx context.Context, h *Handler, requestID string, first, n int) {
	for i := first; i < first+n; i++ {
		_ = h.HandleDrupalCache(ctx, bpfDrupalCacheEvent{
			Type:      EventDrupalCacheRenderArray,
			RequestId: makeRequestID(requestID),
			Caller:    makeCaller(fmt.Sprintf("Drupal\\Core\\Block\\Block%d::build", i)),
			Tags:      makeTags(fmt.Sprintf("node:%d", i)),
			Contexts:  makeContexts("user.permissions"),
			MaxAge:    int64(i),
			Timestamp: uint64(2000 + i*10),
		})
	}
}

// blockingSink holds a trace inside ProcessTrace and measures it on the way in
// and again on the way out. Trace.Drupal is a pointer, so a trace whose state is
// still reachable can be appended to while the sink is reading it.
type blockingSink struct {
	entered chan struct{}
	release chan struct{}

	trace  trace.Trace
	before int
	after  int
}

func newBlockingSink() *blockingSink {
	return &blockingSink{
		entered: make(chan struct{}),
		release: make(chan struct{}),
	}
}

func (s *blockingSink) Initialize() error { return nil }

func (s *blockingSink) ProcessTrace(_ context.Context, t trace.Trace) error {
	s.trace = t
	s.before = cacheEventCount(t)

	close(s.entered)
	<-s.release

	s.after = cacheEventCount(t)

	return nil
}

func cacheEventCount(t trace.Trace) int {
	if t.Drupal == nil {
		return 0
	}

	return len(t.Drupal.CacheEvents)
}

// TestHandler_TraceIsNotMutatedWhileBeingSent shuts a request down while its
// cache events are still arriving.
//
// The handler must take the request out of storage before handing its trace to
// the sink. While the delete was deferred until after the send, those late
// events landed in the trace as it was being serialised.
func TestHandler_TraceIsNotMutatedWhileBeingSent(t *testing.T) {
	ctx := t.Context()

	const (
		collected = 10
		late      = 200
	)

	sink := newBlockingSink()

	h, err := NewHandler(sink, testOptions())
	require.NoError(t, err)

	startRequest(t, h, "req-1")

	functionEvents(ctx, h, "req-1", 5)
	cacheEvents(ctx, h, "req-1", 0, collected)

	var wg sync.WaitGroup

	wg.Add(1)

	go func() {
		defer wg.Done()

		_ = h.Handle(ctx, bpfEvent{
			Type:      EventRequestShutdown,
			RequestId: makeRequestID("req-1"),
			Timestamp: 9000,
		})
	}()

	// Once the sink has the trace, push unseen events at the request behind it.
	<-sink.entered

	cacheEvents(ctx, h, "req-1", collected, late)

	close(sink.release)

	wg.Wait()

	assert.Equal(t, collected, sink.before, "the sink should be handed every event collected before shutdown")
	assert.Equal(t, sink.before, sink.after, "the trace was mutated while the sink held it")

	// Compared as a count: the slice behind it is hundreds of entries of noise.
	assert.Equal(t, collected, cacheEventCount(sink.trace))
}

// TestHandler_ConcurrentEventStreams runs both ring buffer paths at one open
// request at once. The two write disjoint parts of the trace, which is what
// makes a single lock enough; this holds that property under -race.
func TestHandler_ConcurrentEventStreams(t *testing.T) {
	ctx := t.Context()

	const (
		functions = 500
		// Below DefaultMaxCacheEvents, so nothing is dropped.
		cached = 200
	)

	h, sink := newTestHandler(t)

	startRequest(t, h, "req-1")

	var wg sync.WaitGroup

	wg.Add(2)

	go func() {
		defer wg.Done()

		functionEvents(ctx, h, "req-1", functions)
	}()

	go func() {
		defer wg.Done()

		cacheEvents(ctx, h, "req-1", 0, cached)
	}()

	wg.Wait()

	require.NoError(t, h.Handle(ctx, bpfEvent{
		Type:      EventRequestShutdown,
		RequestId: makeRequestID("req-1"),
		Timestamp: 9000,
	}))

	traces := sink.Traces()
	require.Len(t, traces, 1)

	tr := traces[0]

	assert.Equal(t, "req-1", tr.Metadata.ID)
	assert.Len(t, tr.FunctionCalls, functions)

	require.NotNil(t, tr.Drupal)
	assert.Len(t, tr.Drupal.CacheEvents, cached)
	assert.Zero(t, tr.Drupal.CacheEventsDropped)
}

// TestHandler_ConcurrentShutdown races shutdown against events still arriving.
// The counts are a matter of timing, so what is asserted is what holds however
// it lands: a request completes at most once, and no trace grows beyond what
// went into it.
func TestHandler_ConcurrentShutdown(t *testing.T) {
	ctx := t.Context()

	const (
		requests  = 32
		functions = 100
		cached    = 50
	)

	h, sink := newTestHandler(t)

	var wg sync.WaitGroup

	for r := range requests {
		requestID := fmt.Sprintf("req-%d", r)

		startRequest(t, h, requestID)

		wg.Add(3)

		go func() {
			defer wg.Done()

			functionEvents(ctx, h, requestID, functions)
		}()

		go func() {
			defer wg.Done()

			cacheEvents(ctx, h, requestID, 0, cached)
		}()

		go func() {
			defer wg.Done()

			// A failure here is the race landing, not a problem.
			_ = h.Handle(ctx, bpfEvent{
				Type:      EventRequestShutdown,
				RequestId: makeRequestID(requestID),
				Timestamp: 9000,
			})
		}()
	}

	wg.Wait()

	seen := make(map[string]int)

	for _, tr := range sink.Traces() {
		seen[tr.Metadata.ID]++

		assert.LessOrEqual(t, len(tr.FunctionCalls), functions)

		if tr.Drupal != nil {
			assert.LessOrEqual(t, len(tr.Drupal.CacheEvents), cached)
		}
	}

	for id, count := range seen {
		assert.Equalf(t, 1, count, "request %s completed more than once", id)
	}
}

// TestHandler_ConcurrentDistinctRequests is the ordinary case: many requests in
// flight at once, each with events on both ring buffers.
func TestHandler_ConcurrentDistinctRequests(t *testing.T) {
	ctx := t.Context()

	const (
		requests  = 16
		functions = 50
		cached    = 25
	)

	h, sink := newTestHandler(t)

	var wg sync.WaitGroup

	for r := range requests {
		wg.Add(1)

		go func() {
			defer wg.Done()

			requestID := fmt.Sprintf("req-%d", r)

			_ = h.Handle(ctx, bpfEvent{
				Type:      EventRequestInit,
				RequestId: makeRequestID(requestID),
				Uri:       makeURI("/node/1"),
				Method:    makeMethod("GET"),
				Timestamp: 1000,
			})

			var inner sync.WaitGroup

			inner.Add(2)

			go func() {
				defer inner.Done()

				functionEvents(ctx, h, requestID, functions)
			}()

			go func() {
				defer inner.Done()

				cacheEvents(ctx, h, requestID, 0, cached)
			}()

			inner.Wait()

			_ = h.Handle(ctx, bpfEvent{
				Type:      EventRequestShutdown,
				RequestId: makeRequestID(requestID),
				Timestamp: 9000,
			})
		}()
	}

	wg.Wait()

	traces := sink.Traces()
	require.Len(t, traces, requests)

	for _, tr := range traces {
		assert.Len(t, tr.FunctionCalls, functions)

		require.NotNil(t, tr.Drupal)
		assert.Len(t, tr.Drupal.CacheEvents, cached)
	}
}
