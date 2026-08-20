package main

import (
	"context"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skpr/compass/pkg/trace"
)

func TestBroadcaster_Subscribe(t *testing.T) {
	b := NewBroadcaster()

	ch := b.Subscribe()
	defer b.Unsubscribe(ch)

	// Allow the goroutine to process.
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, 1, b.Subscribers())
}

func TestBroadcaster_MultipleSubscribers(t *testing.T) {
	b := NewBroadcaster()

	ch1 := b.Subscribe()
	ch2 := b.Subscribe()
	ch3 := b.Subscribe()

	// Allow the goroutine to process.
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, 3, b.Subscribers())

	b.Unsubscribe(ch1)
	b.Unsubscribe(ch2)
	b.Unsubscribe(ch3)
}

func TestBroadcaster_Unsubscribe(t *testing.T) {
	b := NewBroadcaster()

	ch := b.Subscribe()

	// Allow the goroutine to process.
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 1, b.Subscribers())

	b.Unsubscribe(ch)

	// Allow the goroutine to process.
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 0, b.Subscribers())
}

func TestBroadcaster_ProcessTrace(t *testing.T) {
	b := NewBroadcaster()

	ch := b.Subscribe()
	defer b.Unsubscribe(ch)

	// Allow subscribe to register.
	time.Sleep(50 * time.Millisecond)

	tr := trace.Trace{
		Metadata: trace.Metadata{
			ID:     "test-123",
			Source: trace.SourceHTTP,
		},
	}

	err := b.ProcessTrace(context.Background(), tr)
	require.NoError(t, err)

	select {
	case received := <-ch:
		assert.Equal(t, "test-123", received.Metadata.ID)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for trace")
	}
}

func TestBroadcaster_Broadcast_MultipleSubscribers(t *testing.T) {
	b := NewBroadcaster()

	ch1 := b.Subscribe()
	ch2 := b.Subscribe()
	defer b.Unsubscribe(ch1)
	defer b.Unsubscribe(ch2)

	// Allow subscribes to register.
	time.Sleep(50 * time.Millisecond)

	tr := trace.Trace{
		Metadata: trace.Metadata{
			ID: "multi-test",
		},
	}

	err := b.ProcessTrace(context.Background(), tr)
	require.NoError(t, err)

	for _, ch := range []chan trace.Trace{ch1, ch2} {
		select {
		case received := <-ch:
			assert.Equal(t, "multi-test", received.Metadata.ID)
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for trace")
		}
	}
}

func TestBroadcaster_ResubscribeAfterEmpty(t *testing.T) {
	b := NewBroadcaster()

	ch := b.Subscribe()

	// Allow subscribe to register.
	time.Sleep(50 * time.Millisecond)

	b.Unsubscribe(ch)

	// Allow unsubscribe to register.
	time.Sleep(50 * time.Millisecond)
	require.Equal(t, 0, b.Subscribers())

	// A client which connects after the last one disconnected still receives traces.
	ch = b.Subscribe()
	defer b.Unsubscribe(ch)

	// Allow subscribe to register.
	time.Sleep(50 * time.Millisecond)
	require.Equal(t, 1, b.Subscribers())

	err := b.ProcessTrace(context.Background(), trace.Trace{
		Metadata: trace.Metadata{ID: "resubscribed"},
	})
	require.NoError(t, err)

	select {
	case received := <-ch:
		assert.Equal(t, "resubscribed", received.Metadata.ID)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for trace")
	}
}

func TestBroadcaster_DropsForSlowConsumers(t *testing.T) {
	b := NewBroadcaster()

	ch := b.Subscribe()
	defer b.Unsubscribe(ch)

	// Allow subscribe to register.
	time.Sleep(50 * time.Millisecond)

	before := testutil.ToFloat64(metricTracesDropped)

	// Fill the subscriber buffer (10) and then overflow it without reading.
	for i := 0; i < 15; i++ {
		require.NoError(t, b.ProcessTrace(context.Background(), trace.Trace{
			Metadata: trace.Metadata{ID: "slow"},
		}))
	}

	assert.Greater(t, testutil.ToFloat64(metricTracesDropped), before)
}

func TestBroadcaster_Initialize(t *testing.T) {
	b := NewBroadcaster()
	assert.NoError(t, b.Initialize())
}

func TestBroadcaster_ProcessTrace_ContextCancelled(t *testing.T) {
	b := NewBroadcaster()
	// No subscribers, so broadcast channel will block.

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	tr := trace.Trace{
		Metadata: trace.Metadata{
			ID: "cancelled",
		},
	}

	err := b.ProcessTrace(ctx, tr)
	assert.ErrorIs(t, err, context.Canceled)
}
