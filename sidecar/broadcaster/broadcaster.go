package broadcaster

import (
	"context"
	"sync"

	"github.com/skpr/compass/trace"
)

type Broadcaster struct {
	mu         sync.Mutex
	subs       map[chan trace.Trace]struct{}
	addSub     chan chan trace.Trace
	removeSub  chan chan trace.Trace
	broadcast  chan trace.Trace
	stopSignal chan struct{}
}

// New creates and starts a new broadcaster.
func New() *Broadcaster {
	b := &Broadcaster{
		subs:       make(map[chan trace.Trace]struct{}),
		addSub:     make(chan chan trace.Trace),
		removeSub:  make(chan chan trace.Trace),
		broadcast:  make(chan trace.Trace),
		stopSignal: make(chan struct{}),
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
				default: // prevent blocking on slow consumers
				}
			}
			b.mu.Unlock()

		case sub := <-b.addSub:
			b.mu.Lock()
			b.subs[sub] = struct{}{}
			b.mu.Unlock()

		case sub := <-b.removeSub:
			b.mu.Lock()
			delete(b.subs, sub)
			close(sub)
			if len(b.subs) == 0 {
				select {
				case <-b.stopSignal:
					// already closed
				default:
					close(b.stopSignal)
				}
			}
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
	return len(b.subs)
}

// OnEmpty returns a channel that is closed when all subscribers are gone.
func (b *Broadcaster) OnEmpty() <-chan struct{} {
	return b.stopSignal
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
