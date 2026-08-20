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
}

// NewBroadcaster creates and starts a new broadcaster.
func NewBroadcaster() *Broadcaster {
	b := &Broadcaster{
		subs:      make(map[chan trace.Trace]struct{}),
		addSub:    make(chan chan trace.Trace),
		removeSub: make(chan chan trace.Trace),
		broadcast: make(chan trace.Trace),
	}

	go b.run()

	return b
}

func (b *Broadcaster) run() {
	for {
		select {
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

		case sub := <-b.removeSub:
			b.mu.Lock()
			if _, ok := b.subs[sub]; ok {
				delete(b.subs, sub)
				close(sub)
			}
			b.count.Store(int64(len(b.subs)))
			b.mu.Unlock()
		}
	}
}

// Subscribe registers a new consumer and returns its channel.
func (b *Broadcaster) Subscribe() chan trace.Trace {
	ch := make(chan trace.Trace, 10)
	b.addSub <- ch

	return ch
}

// Unsubscribe removes a consumer.
func (b *Broadcaster) Unsubscribe(ch chan trace.Trace) {
	b.removeSub <- ch
}

// Subscribers returns the number of active subscribers.
func (b *Broadcaster) Subscribers() int {
	return int(b.count.Load())
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
	}
}
