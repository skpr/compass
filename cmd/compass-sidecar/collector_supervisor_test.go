package main

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
)

type testCollectorDemand struct {
	count   atomic.Int64
	changes chan struct{}
}

func newTestCollectorDemand(subscribers int) *testCollectorDemand {
	d := &testCollectorDemand{changes: make(chan struct{}, 1)}
	d.count.Store(int64(subscribers))
	return d
}

func (d *testCollectorDemand) Subscribers() int {
	return int(d.count.Load())
}

func (d *testCollectorDemand) SubscriptionChanges() <-chan struct{} {
	return d.changes
}

func (d *testCollectorDemand) set(subscribers int) {
	d.count.Store(int64(subscribers))
	select {
	case d.changes <- struct{}{}:
	default:
	}
}

func testCollectorSupervisor(demand collectorDemand, run collectorRunner) *collectorSupervisor {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	s := newCollectorSupervisor(logger, demand, run)
	s.setRunning = func(bool) {}
	return s
}

func runCollectorSupervisor(t *testing.T, s *collectorSupervisor) (context.CancelFunc, <-chan error) {
	t.Helper()

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		done <- s.Run(ctx)
	}()

	return cancel, done
}

func waitCollectorSignal(t *testing.T, ch <-chan struct{}, message string) {
	t.Helper()

	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal(message)
	}
}

func waitCollectorSupervisor(t *testing.T, done <-chan error) {
	t.Helper()

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("collector supervisor did not stop")
	}
}

func updateMaximum(maximum *atomic.Int64, value int64) {
	for {
		current := maximum.Load()
		if value <= current || maximum.CompareAndSwap(current, value) {
			return
		}
	}
}

func TestCollectorSupervisor_RapidSubscriberLifecycle(t *testing.T) {
	b := NewBroadcaster()

	started := make(chan struct{}, 128)
	stopped := make(chan struct{}, 128)
	var active atomic.Int64
	var maximum atomic.Int64

	s := testCollectorSupervisor(b, func(ctx context.Context) error {
		nowActive := active.Add(1)
		updateMaximum(&maximum, nowActive)
		started <- struct{}{}

		<-ctx.Done()
		active.Add(-1)
		stopped <- struct{}{}
		return ctx.Err()
	})

	cancel, done := runCollectorSupervisor(t, s)

	for range 100 {
		subscriber := b.Subscribe()
		waitCollectorSignal(t, started, "collector did not start for subscriber")

		b.Unsubscribe(subscriber)
		waitCollectorSignal(t, stopped, "collector did not stop after unsubscribe")
	}

	cancel()
	waitCollectorSupervisor(t, done)

	assert.Zero(t, active.Load())
	assert.Equal(t, int64(1), maximum.Load(), "collectors overlapped")
}

func TestCollectorSupervisor_OnlyStopsForLastSubscriber(t *testing.T) {
	b := NewBroadcaster()
	started := make(chan struct{}, 2)
	stopped := make(chan struct{}, 2)

	s := testCollectorSupervisor(b, func(ctx context.Context) error {
		started <- struct{}{}
		<-ctx.Done()
		stopped <- struct{}{}
		return ctx.Err()
	})

	cancel, done := runCollectorSupervisor(t, s)

	first := b.Subscribe()
	waitCollectorSignal(t, started, "collector did not start")
	second := b.Subscribe()

	b.Unsubscribe(first)
	select {
	case <-stopped:
		t.Fatal("collector stopped while one subscriber remained")
	case <-time.After(25 * time.Millisecond):
	}

	b.Unsubscribe(second)
	waitCollectorSignal(t, stopped, "collector did not stop for final subscriber")

	cancel()
	waitCollectorSupervisor(t, done)
}

func TestCollectorSupervisor_ReconnectWaitsForCancellation(t *testing.T) {
	demand := newTestCollectorDemand(0)
	started := make(chan struct{}, 2)
	cancelled := make(chan struct{}, 1)
	release := make(chan struct{})
	var active atomic.Int64
	var maximum atomic.Int64

	s := testCollectorSupervisor(demand, func(ctx context.Context) error {
		nowActive := active.Add(1)
		updateMaximum(&maximum, nowActive)
		started <- struct{}{}

		<-ctx.Done()
		cancelled <- struct{}{}
		<-release
		active.Add(-1)
		return ctx.Err()
	})

	cancel, done := runCollectorSupervisor(t, s)

	demand.set(1)
	waitCollectorSignal(t, started, "collector did not start")
	demand.set(0)
	waitCollectorSignal(t, cancelled, "collector cancellation was not requested")

	demand.set(1)
	select {
	case <-started:
		t.Fatal("replacement collector overlapped cancellation")
	case <-time.After(25 * time.Millisecond):
	}

	close(release)
	waitCollectorSignal(t, started, "collector did not restart after cancellation completed")

	cancel()
	waitCollectorSupervisor(t, done)

	assert.Zero(t, active.Load())
	assert.Equal(t, int64(1), maximum.Load(), "collectors overlapped")
}

func TestCollectorSupervisor_UsesBoundedExponentialBackoff(t *testing.T) {
	demand := newTestCollectorDemand(1)
	attempts := make(chan time.Time, 8)

	s := testCollectorSupervisor(demand, func(context.Context) error {
		attempts <- time.Now()
		return errors.New("collector failed")
	})
	s.initialBackoff = 15 * time.Millisecond
	s.maxBackoff = 30 * time.Millisecond
	s.backoffReset = time.Hour

	cancel, done := runCollectorSupervisor(t, s)

	times := make([]time.Time, 4)
	for i := range times {
		select {
		case times[i] = <-attempts:
		case <-time.After(time.Second):
			t.Fatal("collector retry did not occur")
		}
	}

	cancel()
	waitCollectorSupervisor(t, done)

	assert.GreaterOrEqual(t, times[1].Sub(times[0]), 10*time.Millisecond)
	assert.GreaterOrEqual(t, times[2].Sub(times[1]), 20*time.Millisecond)
	assert.GreaterOrEqual(t, times[3].Sub(times[2]), 20*time.Millisecond)
	assert.Equal(t, 30*time.Millisecond, growBackoff(30*time.Millisecond, 30*time.Millisecond))
	assert.Equal(t, 30*time.Millisecond, growBackoff(20*time.Millisecond, 30*time.Millisecond))
}

func TestCollectorSupervisor_ZeroDemandCancelsBackoffAndReconnectsImmediately(t *testing.T) {
	demand := newTestCollectorDemand(1)
	attempts := make(chan struct{}, 4)
	states := make(chan bool, 8)

	s := testCollectorSupervisor(demand, func(context.Context) error {
		attempts <- struct{}{}
		return errors.New("collector failed")
	})
	s.initialBackoff = 100 * time.Millisecond
	s.maxBackoff = 200 * time.Millisecond
	s.setRunning = func(running bool) {
		states <- running
	}

	cancel, done := runCollectorSupervisor(t, s)

	waitCollectorSignal(t, attempts, "initial collector attempt did not run")
	require.True(t, <-states)
	require.False(t, <-states)

	demand.set(0)
	select {
	case <-attempts:
		t.Fatal("collector retried without demand")
	case <-time.After(150 * time.Millisecond):
	}

	reconnected := time.Now()
	demand.set(1)
	waitCollectorSignal(t, attempts, "collector did not reconnect")
	assert.Less(t, time.Since(reconnected), 50*time.Millisecond)

	cancel()
	waitCollectorSupervisor(t, done)
}

func TestCollectorSupervisor_ParentCancellationJoinsRunner(t *testing.T) {
	demand := newTestCollectorDemand(1)
	started := make(chan struct{})
	exited := make(chan struct{})
	states := make(chan bool, 2)

	s := testCollectorSupervisor(demand, func(ctx context.Context) error {
		close(started)
		<-ctx.Done()
		time.Sleep(20 * time.Millisecond)
		close(exited)
		return ctx.Err()
	})
	s.setRunning = func(running bool) {
		states <- running
	}

	cancel, done := runCollectorSupervisor(t, s)
	waitCollectorSignal(t, started, "collector did not start")
	cancel()
	waitCollectorSupervisor(t, done)

	select {
	case <-exited:
	default:
		t.Fatal("supervisor returned before its runner exited")
	}

	assert.Equal(t, []bool{true, false}, []bool{<-states, <-states})
}

func TestCollectorSupervisor_ParentCancellationInterruptsBackoff(t *testing.T) {
	demand := newTestCollectorDemand(1)
	attempted := make(chan struct{})
	stopped := make(chan bool, 1)

	s := testCollectorSupervisor(demand, func(context.Context) error {
		close(attempted)
		return errors.New("collector failed")
	})
	s.initialBackoff = time.Hour
	s.maxBackoff = time.Hour
	s.setRunning = func(running bool) {
		if !running {
			stopped <- true
		}
	}

	cancel, done := runCollectorSupervisor(t, s)
	waitCollectorSignal(t, attempted, "collector did not run")

	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("collector failure was not observed")
	}

	cancel()
	waitCollectorSupervisor(t, done)
}
