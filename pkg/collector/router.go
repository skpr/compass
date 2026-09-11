package collector

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/skpr/compass/pkg/trace"
	"github.com/skpr/compass/pkg/tracer/sink"
)

// CollectorFunc runs a collector for a single pod target, writing the traces it
// produces to sink, until ctx is cancelled. It is injected so the routing logic
// can be tested without the eBPF tracer, and so the eBPF-dependent
// implementation stays out of this package.
type CollectorFunc func(ctx context.Context, target Target, sink sink.Interface) error

const (
	defaultBufferSize        = 10
	defaultInitialBackoff    = time.Second
	defaultMaxBackoff        = 30 * time.Second
	defaultBackoffResetAfter = time.Minute
)

// Router fans traces out per pod target. Each distinct target runs exactly one
// collector, started when its first subscriber connects and stopped when its
// last one leaves, and a target's traces reach only the subscribers that asked
// for that target.
type Router struct {
	ctx    context.Context
	run    CollectorFunc
	logger *slog.Logger

	bufferSize     int
	initialBackoff time.Duration
	maxBackoff     time.Duration
	backoffReset   time.Duration
	// maxTargets caps the number of distinct pods collected at once. Zero means
	// unlimited. It bounds the eBPF programs, probes and /proc scans a single
	// daemon can be driven to start by clients naming distinct pod UIDs.
	maxTargets int

	// Hooks for metrics; no-ops by default so the router carries no metrics
	// dependency of its own.
	onCollectorStart func(target Target)
	onCollectorStop  func(target Target)
	onTraceDropped   func(target Target)

	mu      sync.Mutex
	targets map[string]*targetStream
}

// Option configures a Router.
type Option func(*Router)

// WithBackoff sets the collector restart backoff parameters.
func WithBackoff(initial, maximum, resetAfter time.Duration) Option {
	return func(r *Router) {
		r.initialBackoff = initial
		r.maxBackoff = maximum
		r.backoffReset = resetAfter
	}
}

// WithBufferSize sets the per-subscriber channel buffer.
func WithBufferSize(n int) Option {
	return func(r *Router) { r.bufferSize = n }
}

// WithMaxTargets caps the number of distinct pods collected concurrently.
// Non-positive values leave the count unlimited. When the cap is reached,
// CanAccept reports false for a not-yet-collected pod so the daemon can reject
// the request rather than starting another collector.
func WithMaxTargets(n int) Option {
	return func(r *Router) { r.maxTargets = n }
}

// WithMetrics installs callbacks invoked when a collector starts or stops and
// when a trace is dropped for a slow subscriber.
func WithMetrics(onStart, onStop, onDropped func(Target)) Option {
	return func(r *Router) {
		if onStart != nil {
			r.onCollectorStart = onStart
		}
		if onStop != nil {
			r.onCollectorStop = onStop
		}
		if onDropped != nil {
			r.onTraceDropped = onDropped
		}
	}
}

// NewRouter creates a Router. When ctx is cancelled every collector is stopped
// and every subscriber channel is closed, so blocked readers unblock.
func NewRouter(ctx context.Context, logger *slog.Logger, run CollectorFunc, opts ...Option) *Router {
	r := &Router{
		ctx:              ctx,
		run:              run,
		logger:           logger,
		bufferSize:       defaultBufferSize,
		initialBackoff:   defaultInitialBackoff,
		maxBackoff:       defaultMaxBackoff,
		backoffReset:     defaultBackoffResetAfter,
		onCollectorStart: func(Target) {},
		onCollectorStop:  func(Target) {},
		onTraceDropped:   func(Target) {},
		targets:          make(map[string]*targetStream),
	}

	for _, opt := range opts {
		opt(r)
	}

	go r.awaitShutdown()

	return r
}

func (r *Router) awaitShutdown() {
	<-r.ctx.Done()

	r.mu.Lock()
	streams := make([]*targetStream, 0, len(r.targets))
	for _, ts := range r.targets {
		streams = append(streams, ts)
	}
	r.targets = make(map[string]*targetStream)
	r.mu.Unlock()

	for _, ts := range streams {
		ts.shutdown()
	}
}

// Subscribe registers a consumer for a target's traces and returns its channel
// along with an idempotent unsubscribe function. The first subscriber for a
// target starts its collector.
func (r *Router) Subscribe(target Target) (<-chan trace.Trace, func()) {
	key := target.Key()
	ch := make(chan trace.Trace, r.bufferSize)

	for {
		r.mu.Lock()

		if r.ctx.Err() != nil {
			// The router is shutting down; hand back a closed channel so the
			// caller's stream ends immediately rather than blocking forever.
			r.mu.Unlock()
			close(ch)

			return ch, func() {}
		}

		ts, ok := r.targets[key]
		if !ok {
			ts = r.startTarget(target)
			r.targets[key] = ts
		}

		// A target being torn down stays in the map until its collector has
		// stopped. Wait for that to finish, then loop and create a fresh one so
		// two collectors never run for the same pod at once.
		if gone, waiting := ts.addSubscriber(ch); waiting {
			r.mu.Unlock()
			<-gone

			continue
		}

		r.mu.Unlock()

		var once sync.Once

		return ch, func() {
			once.Do(func() { r.unsubscribe(key, ts, ch) })
		}
	}
}

func (r *Router) startTarget(target Target) *targetStream {
	ts := &targetStream{
		target:         target,
		logger:         r.logger,
		bufferSize:     r.bufferSize,
		initialBackoff: r.initialBackoff,
		maxBackoff:     r.maxBackoff,
		backoffReset:   r.backoffReset,
		onDropped:      r.onTraceDropped,
		subs:           make(map[chan trace.Trace]struct{}),
		runnerDone:     make(chan struct{}),
		gone:           make(chan struct{}),
	}

	ts.ctx, ts.cancel = context.WithCancel(r.ctx)

	r.onCollectorStart(target)

	go func() {
		ts.serve(r.run)
		r.onCollectorStop(target)
	}()

	return ts
}

func (r *Router) unsubscribe(key string, ts *targetStream, ch chan trace.Trace) {
	last := ts.removeSubscriber(ch)
	if !last {
		return
	}

	// Last subscriber for this target: stop the collector and wait for it to
	// exit before removing the target, so a re-subscribe blocks rather than
	// overlapping a new collector with the draining one.
	ts.cancel()
	<-ts.runnerDone

	r.mu.Lock()
	if r.targets[key] == ts {
		delete(r.targets, key)
	}
	r.mu.Unlock()

	close(ts.gone)
}

// Targets returns the number of distinct pods currently being collected. Used
// by tests and metrics.
func (r *Router) Targets() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	return len(r.targets)
}

// CanAccept reports whether a subscription for target may proceed under the
// configured target ceiling. A pod already being collected always may (it adds
// a subscriber, not a collector); a new pod may only when the ceiling has room.
//
// This is an admission check, not a reservation: it is not atomic with the
// Subscribe that follows, so a burst of concurrent subscriptions for distinct
// new pods can transiently exceed maxTargets by the size of that burst. The cap
// is a safety ceiling against unbounded growth, not an exact quota, so that
// relaxation is intentional.
func (r *Router) CanAccept(target Target) bool {
	if r.maxTargets <= 0 {
		return true
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.targets[target.Key()]; ok {
		return true
	}

	return len(r.targets) < r.maxTargets
}

// targetStream owns the subscribers and the collector for one pod target. It is
// the sink.Interface the collector writes into.
type targetStream struct {
	target Target
	logger *slog.Logger

	bufferSize     int
	initialBackoff time.Duration
	maxBackoff     time.Duration
	backoffReset   time.Duration
	onDropped      func(Target)

	ctx    context.Context
	cancel context.CancelFunc

	// runnerDone is closed when the collector goroutine has fully exited.
	runnerDone chan struct{}
	// gone is closed once the target has been removed from the router, so a
	// same-target subscribe that raced the teardown can proceed.
	gone chan struct{}

	mu      sync.Mutex
	subs    map[chan trace.Trace]struct{}
	count   int
	closing bool
	closed  bool
}

// addSubscriber registers ch unless the stream is being torn down, in which
// case it returns the gone channel and waiting=true so the caller can wait and
// retry against a fresh stream.
func (ts *targetStream) addSubscriber(ch chan trace.Trace) (gone <-chan struct{}, waiting bool) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if ts.closing {
		return ts.gone, true
	}

	ts.subs[ch] = struct{}{}
	ts.count++

	return nil, false
}

// removeSubscriber drops ch and reports whether it was the last one, marking
// the stream as closing when so.
func (ts *targetStream) removeSubscriber(ch chan trace.Trace) (last bool) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if _, ok := ts.subs[ch]; ok {
		delete(ts.subs, ch)
		ts.count--
	}

	if ts.count <= 0 && !ts.closing {
		ts.closing = true

		return true
	}

	return false
}

// Initialize satisfies sink.Interface.
func (ts *targetStream) Initialize() error { return nil }

// ProcessTrace fans a trace out to the target's subscribers, dropping for any
// whose buffer is full rather than blocking the collector.
func (ts *targetStream) ProcessTrace(ctx context.Context, t trace.Trace) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if ts.closed {
		return nil
	}

	for ch := range ts.subs {
		send(ch, t, func() { ts.onDropped(ts.target) })
	}

	return nil
}

// send a trace to a subscriber which may not be keeping up.
//
// A full buffer means the subscriber is behind, and the question is which
// trace to lose. It used to be the arriving one, which is the wrong end: this
// is a live view of what an application is doing now, and the newest trace is
// the one somebody is waiting to see. So the oldest waiting trace is dropped
// to make room, and the new one takes its place.
//
// A subscriber which is not reading at all still ends up dropping the
// arriving trace, because the room made for it is taken by the time the send
// is attempted. Either way the drop is counted, so a consumer which cannot
// keep up stays visible.
func send(ch chan trace.Trace, t trace.Trace, dropped func()) {
	select {
	case ch <- t:
		return
	default:
	}

	// Make room by taking the oldest, which the subscriber has not read.
	select {
	case <-ch:
	default:
	}

	dropped()

	select {
	case ch <- t:
	default:
	}
}

// serve runs the collector, restarting it with a bounded backoff if it exits
// unexpectedly while the target still has subscribers.
func (ts *targetStream) serve(run CollectorFunc) {
	defer close(ts.runnerDone)

	backoff := ts.initialBackoff

	for {
		started := time.Now()
		err := run(ts.ctx, ts.target, ts)

		if ts.ctx.Err() != nil {
			return
		}

		if time.Since(started) >= ts.backoffReset {
			backoff = ts.initialBackoff
		}

		delay := backoff
		backoff = growBackoff(backoff, ts.maxBackoff)

		if err != nil {
			ts.logger.Error("Collector exited; scheduling restart", "uid", ts.target.UID, "error", err, "retry_in", delay)
		} else {
			ts.logger.Warn("Collector exited; scheduling restart", "uid", ts.target.UID, "retry_in", delay)
		}

		select {
		case <-ts.ctx.Done():
			return
		case <-time.After(delay):
		}
	}
}

// shutdown stops the collector and closes every subscriber channel. Used on
// router shutdown so blocked readers unblock.
func (ts *targetStream) shutdown() {
	ts.cancel()
	<-ts.runnerDone

	ts.mu.Lock()
	defer ts.mu.Unlock()

	if ts.closed {
		return
	}

	ts.closed = true

	for ch := range ts.subs {
		delete(ts.subs, ch)
		close(ch)
	}
}

func growBackoff(current, maximum time.Duration) time.Duration {
	if current >= maximum || current > maximum/2 {
		return maximum
	}

	return current * 2
}
