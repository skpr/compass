package fpm

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skpr/compass/pkg/trace"
)

func makeCaller(caller string) [256]uint8 {
	var arr [256]uint8
	copy(arr[:], caller)
	return arr
}

func makeObjectType(objectType string) [256]uint8 {
	var arr [256]uint8
	copy(arr[:], objectType)
	return arr
}

func makeTags(tags string) [1024]uint8 {
	var arr [1024]uint8
	copy(arr[:], tags)
	return arr
}

func makeContexts(contexts string) [512]uint8 {
	var arr [512]uint8
	copy(arr[:], contexts)
	return arr
}

// startRequest opens a request so that cache events have somewhere to land.
func startRequest(t *testing.T, h *Handler, requestID string) {
	t.Helper()

	require.NoError(t, h.Handle(bpfEvent{
		Type:      EventRequestInit,
		RequestId: makeRequestID(requestID),
		Uri:       makeURI("/node/1"),
		Method:    makeMethod("GET"),
		Timestamp: 1000,
	}))
}

func TestHandler_HandleDrupalCache_EmptyRequestID(t *testing.T) {
	h, _ := newTestHandler(t)

	err := h.HandleDrupalCache(bpfDrupalCacheEvent{
		Type: EventDrupalCacheRenderArray,
	})
	assert.ErrorContains(t, err, "empty request id")
}

func TestHandler_HandleDrupalCache_UnknownType(t *testing.T) {
	h, _ := newTestHandler(t)

	startRequest(t, h, "req-1")

	err := h.HandleDrupalCache(bpfDrupalCacheEvent{
		Type:      99,
		RequestId: makeRequestID("req-1"),
	})
	assert.ErrorContains(t, err, "unknown drupal cache event type")
}

func TestHandler_HandleDrupalCache_NotFound(t *testing.T) {
	h, _ := newTestHandler(t)

	err := h.HandleDrupalCache(bpfDrupalCacheEvent{
		Type:      EventDrupalCacheRenderArray,
		RequestId: makeRequestID("req-nonexistent"),
	})
	assert.ErrorContains(t, err, "not found in storage")
}

func TestHandler_HandleDrupalCache_RenderArray(t *testing.T) {
	h, _ := newTestHandler(t)

	startRequest(t, h, "req-1")

	require.NoError(t, h.HandleDrupalCache(bpfDrupalCacheEvent{
		Type:      EventDrupalCacheRenderArray,
		RequestId: makeRequestID("req-1"),
		Caller:    makeCaller(`Drupal\Core\Render\Renderer::doRender`),
		MaxAge:    0,
		Tags:      makeTags("node:1 config:system.site"),
		Contexts:  makeContexts("url.path user.roles"),
		Timestamp: 1500,
	}))

	stored := h.storage.Items()["req-1"].Object.(*state).trace

	require.NotNil(t, stored.Drupal)
	require.Len(t, stored.Drupal.CacheEvents, 1)

	event := stored.Drupal.CacheEvents[0]
	assert.Equal(t, trace.CacheOriginRenderArray, event.Origin)
	assert.Equal(t, `Drupal\Core\Render\Renderer::doRender`, event.Caller)
	assert.Empty(t, event.ObjectType)
	assert.Equal(t, int64(0), event.MaxAge)
	assert.Equal(t, []string{"node:1", "config:system.site"}, event.Tags)
	assert.Equal(t, []string{"url.path", "user.roles"}, event.Contexts)
	assert.Equal(t, int64(1500), event.StartTime)
	assert.Equal(t, int64(1), event.Calls)
}

func TestHandler_HandleDrupalCache_Object(t *testing.T) {
	h, _ := newTestHandler(t)

	startRequest(t, h, "req-1")

	require.NoError(t, h.HandleDrupalCache(bpfDrupalCacheEvent{
		Type:       EventDrupalCacheObject,
		RequestId:  makeRequestID("req-1"),
		Caller:     makeCaller(`Drupal\node\NodeViewBuilder::build`),
		MaxAge:     3600,
		ObjectType: makeObjectType(`Drupal\node\Entity\Node`),
		Tags:       makeTags("node:1"),
		Timestamp:  1600,
	}))

	stored := h.storage.Items()["req-1"].Object.(*state).trace

	require.Len(t, stored.Drupal.CacheEvents, 1)

	event := stored.Drupal.CacheEvents[0]
	assert.Equal(t, trace.CacheOriginObject, event.Origin)
	assert.Equal(t, `Drupal\node\Entity\Node`, event.ObjectType)
	assert.Equal(t, int64(3600), event.MaxAge)
	assert.Equal(t, []string{"node:1"}, event.Tags)
	assert.Nil(t, event.Contexts)
}

func TestHandler_HandleDrupalCache_IdenticalEventsAggregate(t *testing.T) {
	h, _ := newTestHandler(t)

	startRequest(t, h, "req-1")

	event := bpfDrupalCacheEvent{
		Type:      EventDrupalCacheRenderArray,
		RequestId: makeRequestID("req-1"),
		Caller:    makeCaller(`Drupal\Core\Render\Renderer::doRender`),
		MaxAge:    -1,
		Tags:      makeTags("node:1"),
		Timestamp: 1500,
	}

	for range 500 {
		require.NoError(t, h.HandleDrupalCache(event))
	}

	// A later, different event must not fold into the first.
	other := event
	other.MaxAge = 0
	other.Timestamp = 9000
	require.NoError(t, h.HandleDrupalCache(other))

	stored := h.storage.Items()["req-1"].Object.(*state).trace

	require.Len(t, stored.Drupal.CacheEvents, 2)
	assert.Equal(t, int64(500), stored.Drupal.CacheEvents[0].Calls)
	// The retained start time is that of the first event of its kind.
	assert.Equal(t, int64(1500), stored.Drupal.CacheEvents[0].StartTime)
	assert.Equal(t, int64(1), stored.Drupal.CacheEvents[1].Calls)
	assert.Equal(t, int64(9000), stored.Drupal.CacheEvents[1].StartTime)
	assert.Zero(t, stored.Drupal.CacheEventsDropped)
}

func TestHandler_HandleDrupalCache_OriginIsPartOfTheKey(t *testing.T) {
	h, _ := newTestHandler(t)

	startRequest(t, h, "req-1")

	event := bpfDrupalCacheEvent{
		Type:      EventDrupalCacheRenderArray,
		RequestId: makeRequestID("req-1"),
		Caller:    makeCaller("Same::caller"),
		MaxAge:    -1,
		Tags:      makeTags("node:1"),
	}
	require.NoError(t, h.HandleDrupalCache(event))

	event.Type = EventDrupalCacheObject
	require.NoError(t, h.HandleDrupalCache(event))

	stored := h.storage.Items()["req-1"].Object.(*state).trace
	assert.Len(t, stored.Drupal.CacheEvents, 2)
}

func TestHandler_HandleDrupalCache_DistinctEventsAreCapped(t *testing.T) {
	sink := &mockSink{}

	h, err := NewHandler(sink, Options{Expire: time.Minute, MaxCacheEvents: 3})
	require.NoError(t, err)

	startRequest(t, h, "req-1")

	for i := range 10 {
		require.NoError(t, h.HandleDrupalCache(bpfDrupalCacheEvent{
			Type:      EventDrupalCacheRenderArray,
			RequestId: makeRequestID("req-1"),
			Caller:    makeCaller(fmt.Sprintf("Caller%d::render", i)),
			MaxAge:    0,
		}))
	}

	stored := h.storage.Items()["req-1"].Object.(*state).trace

	assert.Len(t, stored.Drupal.CacheEvents, 3)
	assert.Equal(t, 7, stored.Drupal.CacheEventsDropped)
}

func TestHandler_HandleDrupalCache_CappedEventsStillAggregate(t *testing.T) {
	sink := &mockSink{}

	h, err := NewHandler(sink, Options{Expire: time.Minute, MaxCacheEvents: 1})
	require.NoError(t, err)

	startRequest(t, h, "req-1")

	kept := bpfDrupalCacheEvent{
		Type:      EventDrupalCacheRenderArray,
		RequestId: makeRequestID("req-1"),
		Caller:    makeCaller("Kept::render"),
		MaxAge:    0,
	}

	require.NoError(t, h.HandleDrupalCache(kept))
	require.NoError(t, h.HandleDrupalCache(kept))

	stored := h.storage.Items()["req-1"].Object.(*state).trace

	require.Len(t, stored.Drupal.CacheEvents, 1)
	assert.Equal(t, int64(2), stored.Drupal.CacheEvents[0].Calls)
	assert.Zero(t, stored.Drupal.CacheEventsDropped)
}

func TestHandler_HandleDrupalCache_ReachesTheSink(t *testing.T) {
	h, sink := newTestHandler(t)

	startRequest(t, h, "req-1")

	require.NoError(t, h.Handle(bpfEvent{
		Type:         EventFunction,
		RequestId:    makeRequestID("req-1"),
		FunctionName: makeFunctionName("handler"),
		Timestamp:    1500,
		Elapsed:      300,
		Memory:       2048,
	}))

	require.NoError(t, h.HandleDrupalCache(bpfDrupalCacheEvent{
		Type:      EventDrupalCacheRenderArray,
		RequestId: makeRequestID("req-1"),
		Caller:    makeCaller("Blocker::render"),
		MaxAge:    0,
		Timestamp: 1600,
	}))

	require.NoError(t, h.Handle(bpfEvent{
		Type:      EventRequestShutdown,
		RequestId: makeRequestID("req-1"),
		Timestamp: 2000,
	}))

	require.Len(t, sink.traces, 1)
	require.NotNil(t, sink.traces[0].Drupal)
	assert.Len(t, sink.traces[0].Drupal.CacheEvents, 1)
}

// A request with cache events but no function calls is still worth reporting:
// a high function threshold can filter out every function while Drupal keeps
// deriving cacheability.
func TestHandler_HandleDrupalCache_TraceWithoutFunctionsIsSent(t *testing.T) {
	h, sink := newTestHandler(t)

	startRequest(t, h, "req-1")

	require.NoError(t, h.HandleDrupalCache(bpfDrupalCacheEvent{
		Type:      EventDrupalCacheRenderArray,
		RequestId: makeRequestID("req-1"),
		Caller:    makeCaller("Blocker::render"),
		MaxAge:    0,
	}))

	require.NoError(t, h.Handle(bpfEvent{
		Type:      EventRequestShutdown,
		RequestId: makeRequestID("req-1"),
		Timestamp: 2000,
	}))

	require.Len(t, sink.traces, 1)
	assert.Empty(t, sink.traces[0].FunctionCalls)
}

func TestHandler_HandleDrupalCache_NonDrupalTraceHasNoDrupalData(t *testing.T) {
	h, sink := newTestHandler(t)

	startRequest(t, h, "req-1")

	require.NoError(t, h.Handle(bpfEvent{
		Type:         EventFunction,
		RequestId:    makeRequestID("req-1"),
		FunctionName: makeFunctionName("handler"),
		Timestamp:    1500,
		Elapsed:      300,
	}))

	require.NoError(t, h.Handle(bpfEvent{
		Type:      EventRequestShutdown,
		RequestId: makeRequestID("req-1"),
		Timestamp: 2000,
	}))

	require.Len(t, sink.traces, 1)
	assert.Nil(t, sink.traces[0].Drupal)
}
