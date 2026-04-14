package main

import (
	"context"
	"testing"
	"time"

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

func TestBroadcaster_OnEmpty(t *testing.T) {
	b := NewBroadcaster()

	ch := b.Subscribe()

	// Allow subscribe to register.
	time.Sleep(50 * time.Millisecond)

	b.Unsubscribe(ch)

	select {
	case <-b.OnEmpty():
		// Success - channel was closed.
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for OnEmpty signal")
	}
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
