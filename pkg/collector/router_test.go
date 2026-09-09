package collector

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skpr/compass/pkg/trace"
	"github.com/skpr/compass/pkg/tracer/sink"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func mustTarget(t *testing.T, uid string) Target {
	t.Helper()
	return Target{UID: uid}
}

func recv[T any](t *testing.T, ch <-chan T, msg string) T {
	t.Helper()

	select {
	case v := <-ch:
		return v
	case <-time.After(2 * time.Second):
		t.Fatal(msg)
	}

	var zero T
	return zero
}

func expectNothing[T any](t *testing.T, ch <-chan T, within time.Duration, msg string) {
	t.Helper()

	select {
	case <-ch:
		t.Fatal(msg)
	case <-time.After(within):
	}
}

// fakeCollector records lifecycle and exposes the sink each run received so a
// test can push traces through it.
type fakeCollector struct {
	started chan Target
	stopped chan Target
	sinks   chan sink.Interface
	active  atomic.Int64
	maximum atomic.Int64
}

func newFakeCollector() *fakeCollector {
	return &fakeCollector{
		started: make(chan Target, 16),
		stopped: make(chan Target, 16),
		sinks:   make(chan sink.Interface, 16),
	}
}

func (f *fakeCollector) run(ctx context.Context, target Target, s sink.Interface) error {
	now := f.active.Add(1)
	for {
		max := f.maximum.Load()
		if now <= max || f.maximum.CompareAndSwap(max, now) {
			break
		}
	}

	// Non-blocking so a test that ignores one of these channels never stalls
	// the collector and prevents it from observing cancellation.
	send := func(ch chan Target) {
		select {
		case ch <- target:
		default:
		}
	}

	send(f.started)
	select {
	case f.sinks <- s:
	default:
	}

	<-ctx.Done()

	f.active.Add(-1)
	send(f.stopped)

	return ctx.Err()
}

func TestRouter_StartsOneCollectorPerTargetAndFansOut(t *testing.T) {
	fake := newFakeCollector()
	r := NewRouter(t.Context(), testLogger(), fake.run)

	target := mustTarget(t, "12345678-1234-1234-1234-123456789abc")

	ch1, unsub1 := r.Subscribe(target)
	require.Equal(t, target, recv(t, fake.started, "first subscriber did not start a collector"))
	s := recv(t, fake.sinks, "collector did not expose its sink")

	// A second subscriber for the same pod must not start a second collector.
	ch2, unsub2 := r.Subscribe(target)
	expectNothing(t, fake.started, 100*time.Millisecond, "second subscriber started a duplicate collector")

	assert.Equal(t, 1, r.Targets())

	tr := trace.Trace{Metadata: trace.Metadata{ID: "abc"}}
	require.NoError(t, s.ProcessTrace(t.Context(), tr))

	assert.Equal(t, "abc", recv(t, ch1, "subscriber 1 got no trace").Metadata.ID)
	assert.Equal(t, "abc", recv(t, ch2, "subscriber 2 got no trace").Metadata.ID)

	unsub1()
	unsub2()

	assert.Equal(t, target, recv(t, fake.stopped, "collector did not stop after last unsubscribe"))
	assert.Eventually(t, func() bool { return r.Targets() == 0 }, time.Second, 10*time.Millisecond)
}

func TestRouter_IsolatesTargets(t *testing.T) {
	fake := newFakeCollector()
	r := NewRouter(t.Context(), testLogger(), fake.run)

	a := mustTarget(t, "aaaaaaaa-1234-1234-1234-123456789abc")
	b := mustTarget(t, "bbbbbbbb-1234-1234-1234-123456789abc")

	chA, unsubA := r.Subscribe(a)
	recv(t, fake.started, "collector A did not start")
	sinkA := recv(t, fake.sinks, "collector A sink")

	chB, unsubB := r.Subscribe(b)
	recv(t, fake.started, "collector B did not start")
	sinkB := recv(t, fake.sinks, "collector B sink")

	defer unsubA()
	defer unsubB()

	assert.Equal(t, 2, r.Targets(), "distinct pods should run concurrent collectors")

	require.NoError(t, sinkA.ProcessTrace(t.Context(), trace.Trace{Metadata: trace.Metadata{ID: "from-a"}}))

	assert.Equal(t, "from-a", recv(t, chA, "A subscriber missed its trace").Metadata.ID)
	expectNothing(t, chB, 100*time.Millisecond, "A's trace leaked to B's subscriber")

	require.NoError(t, sinkB.ProcessTrace(t.Context(), trace.Trace{Metadata: trace.Metadata{ID: "from-b"}}))
	assert.Equal(t, "from-b", recv(t, chB, "B subscriber missed its trace").Metadata.ID)
}

func TestRouter_StopsOnlyWhenLastSubscriberLeaves(t *testing.T) {
	fake := newFakeCollector()
	r := NewRouter(t.Context(), testLogger(), fake.run)

	target := mustTarget(t, "12345678-1234-1234-1234-123456789abc")

	_, unsub1 := r.Subscribe(target)
	recv(t, fake.started, "collector did not start")
	_, unsub2 := r.Subscribe(target)

	unsub1()
	expectNothing(t, fake.stopped, 100*time.Millisecond, "collector stopped while a subscriber remained")

	unsub2()
	recv(t, fake.stopped, "collector did not stop for the final subscriber")
}

func TestRouter_UnsubscribeIsIdempotent(t *testing.T) {
	fake := newFakeCollector()
	r := NewRouter(t.Context(), testLogger(), fake.run)

	target := mustTarget(t, "12345678-1234-1234-1234-123456789abc")

	_, unsub := r.Subscribe(target)
	recv(t, fake.started, "collector did not start")

	unsub()
	unsub() // must not panic or double-stop
	recv(t, fake.stopped, "collector did not stop")
	expectNothing(t, fake.stopped, 100*time.Millisecond, "collector stopped twice")
}

func TestRouter_ResubscribeAfterTeardownDoesNotOverlap(t *testing.T) {
	fake := newFakeCollector()
	r := NewRouter(t.Context(), testLogger(), fake.run)

	target := mustTarget(t, "12345678-1234-1234-1234-123456789abc")

	for i := 0; i < 20; i++ {
		_, unsub := r.Subscribe(target)
		recv(t, fake.started, "collector did not start")
		unsub()
		recv(t, fake.stopped, "collector did not stop")
	}

	assert.Equal(t, int64(1), fake.maximum.Load(), "collectors overlapped for the same pod")
	assert.Zero(t, fake.active.Load())
}

func TestRouter_DropsForSlowSubscriber(t *testing.T) {
	fake := newFakeCollector()

	var dropped atomic.Int64
	r := NewRouter(t.Context(), testLogger(), fake.run,
		WithBufferSize(2),
		WithMetrics(nil, nil, func(Target) { dropped.Add(1) }),
	)

	target := mustTarget(t, "12345678-1234-1234-1234-123456789abc")

	_, unsub := r.Subscribe(target)
	defer unsub()
	recv(t, fake.started, "collector did not start")
	s := recv(t, fake.sinks, "collector sink")

	// Overflow the buffer (2) without reading.
	for i := 0; i < 10; i++ {
		require.NoError(t, s.ProcessTrace(t.Context(), trace.Trace{Metadata: trace.Metadata{ID: "x"}}))
	}

	assert.Positive(t, dropped.Load(), "expected drops for a subscriber that never reads")
}

func TestRouter_RestartsCollectorOnUnexpectedExit(t *testing.T) {
	started := make(chan struct{}, 8)
	var attempts atomic.Int64

	run := func(ctx context.Context, _ Target, _ sink.Interface) error {
		started <- struct{}{}
		if attempts.Add(1) <= 2 {
			// Exit unexpectedly the first two times.
			return errors.New("boom")
		}
		<-ctx.Done()
		return ctx.Err()
	}

	r := NewRouter(t.Context(), testLogger(), run, WithBackoff(5*time.Millisecond, 10*time.Millisecond, time.Hour))

	target := mustTarget(t, "12345678-1234-1234-1234-123456789abc")

	_, unsub := r.Subscribe(target)
	defer unsub()

	// Three starts: initial + two restarts after the failures.
	for i := 0; i < 3; i++ {
		recv(t, started, "collector did not (re)start after unexpected exit")
	}
}

func TestRouter_ShutdownClosesSubscribers(t *testing.T) {
	fake := newFakeCollector()
	ctx, cancel := context.WithCancel(context.Background())
	r := NewRouter(ctx, testLogger(), fake.run)

	target := mustTarget(t, "12345678-1234-1234-1234-123456789abc")

	ch, _ := r.Subscribe(target)
	recv(t, fake.started, "collector did not start")

	cancel()

	// The subscriber channel is closed on shutdown, so a blocked reader unblocks.
	select {
	case _, ok := <-ch:
		assert.False(t, ok, "subscriber channel should be closed on shutdown")
	case <-time.After(time.Second):
		t.Fatal("subscriber was not closed on shutdown")
	}

	// Subscribing after shutdown returns a closed channel rather than blocking.
	late, _ := r.Subscribe(target)
	_, ok := <-late
	assert.False(t, ok, "subscribe after shutdown should return a closed channel")
}

func TestGrowBackoff(t *testing.T) {
	assert.Equal(t, 2*time.Second, growBackoff(time.Second, 30*time.Second))
	assert.Equal(t, 30*time.Second, growBackoff(20*time.Second, 30*time.Second))
	assert.Equal(t, 30*time.Second, growBackoff(30*time.Second, 30*time.Second))
}
