package main

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/skpr/compass/pkg/trace"
)

// Broadcaster fans traces out to all subscribed streams.
type Broadcaster struct {
	mu    sync.Mutex
	subs  map[chan trace.Trace]struct{}
	count atomic.Int64

	addSub    chan chan trace.Trace
	removeSub chan chan trace.Trace
	broadcast chan trace.Trace
	changes   chan struct{}
	// done is closed when run has returned, so the public methods stop blocking
	// on a loop which is no longer reading.
	done chan struct{}
}

// NewBroadcaster creates and starts a new broadcaster. It runs until ctx is
// cancelled, at which point it closes every subscriber channel so blocked
// consumers unblock and return.
func NewBroadcaster(ctx context.Context) *Broadcaster {
	b := &Broadcaster{
		subs:      make(map[chan trace.Trace]struct{}),
		addSub:    make(chan chan trace.Trace),
		removeSub: make(chan chan trace.Trace),
		broadcast: make(chan trace.Trace),
		changes:   make(chan struct{}, 1),
		done:      make(chan struct{}),
	}

	go b.run(ctx)

	return b
}

func (b *Broadcaster) run(ctx context.Context) {
	defer close(b.done)

	for {
		select {
		case <-ctx.Done():
			b.mu.Lock()
			for ch := range b.subs {
				delete(b.subs, ch)
				close(ch)
			}
			b.count.Store(0)
			b.mu.Unlock()

			return

		case msg := <-b.broadcast:
			b.mu.Lock()
			for ch := range b.subs {
				select {
				case ch <- msg:
				default:
					// Prevent blocking on slow consumers. Dropping is tracked so
					// that a consumer which cannot keep up is observable.
					metricTracesDropped.Inc()
				}
			}
			b.mu.Unlock()

		case sub := <-b.addSub:
			b.mu.Lock()
			b.subs[sub] = struct{}{}
			b.count.Store(int64(len(b.subs)))
			b.mu.Unlock()
			b.notifySubscriptionChange()

		case sub := <-b.removeSub:
			b.mu.Lock()
			if _, ok := b.subs[sub]; ok {
				delete(b.subs, sub)
				close(sub)
				b.count.Store(int64(len(b.subs)))
				b.mu.Unlock()
				b.notifySubscriptionChange()
				continue
			}
			b.mu.Unlock()
		}
	}
}

// Subscribe registers a new consumer and returns its channel. If the
// broadcaster is already shutting down, it returns a closed channel so the
// caller sees the stream end immediately rather than blocking.
func (b *Broadcaster) Subscribe() chan trace.Trace {
	ch := make(chan trace.Trace, 10)

	select {
	case b.addSub <- ch:
		return ch
	case <-b.done:
		close(ch)
		return ch
	}
}

// Unsubscribe removes a consumer. It is a no-op once the broadcaster has shut
// down, since every subscriber channel was closed then.
func (b *Broadcaster) Unsubscribe(ch chan trace.Trace) {
	select {
	case b.removeSub <- ch:
	case <-b.done:
	}
}

// Subscribers returns the number of active subscribers.
func (b *Broadcaster) Subscribers() int {
	return int(b.count.Load())
}

// SubscriptionChanges is notified after the subscriber count changes. Signals
// are coalesced because consumers read the current atomic count after waking.
func (b *Broadcaster) SubscriptionChanges() <-chan struct{} {
	return b.changes
}

func (b *Broadcaster) notifySubscriptionChange() {
	select {
	case b.changes <- struct{}{}:
	default:
	}
}

// Initialize the plugin.
func (b *Broadcaster) Initialize() error {
	return nil
}

// ProcessTrace from the collector.
func (b *Broadcaster) ProcessTrace(ctx context.Context, t trace.Trace) error {
	select {
	case b.broadcast <- t:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-b.done:
		return context.Canceled
	}
}
