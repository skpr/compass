package main

import (
	"context"
	"errors"
	"log/slog"
	"time"
)

const (
	collectorInitialBackoff = time.Second
	collectorMaxBackoff     = 30 * time.Second
	collectorBackoffReset   = time.Minute
)

type collectorDemand interface {
	Subscribers() int
	SubscriptionChanges() <-chan struct{}
}

type collectorRunner func(context.Context) error

type collectorRunResult struct {
	err      error
	duration time.Duration
}

// collectorSupervisor is the sole owner of collector lifecycle state. Demand
// changes, runner exits, retries, and shutdown all enter this one event loop so
// a stale cancellation function cannot race a replacement collector.
type collectorSupervisor struct {
	demand collectorDemand
	run    collectorRunner
	logger *slog.Logger

	initialBackoff time.Duration
	maxBackoff     time.Duration
	backoffReset   time.Duration
	now            func() time.Time
	setRunning     func(bool)
}

func newCollectorSupervisor(logger *slog.Logger, demand collectorDemand, run collectorRunner) *collectorSupervisor {
	return &collectorSupervisor{
		demand:         demand,
		run:            run,
		logger:         logger,
		initialBackoff: collectorInitialBackoff,
		maxBackoff:     collectorMaxBackoff,
		backoffReset:   collectorBackoffReset,
		now:            time.Now,
		setRunning: func(running bool) {
			if running {
				metricCollectorRunning.Set(1)
				return
			}

			metricCollectorRunning.Set(0)
		},
	}
}

// Run supervises the collector until the parent context is cancelled.
func (s *collectorSupervisor) Run(ctx context.Context) error {
	changes := s.demand.SubscriptionChanges()
	backoff := s.initialBackoff

	var (
		runCancel     context.CancelFunc
		runDone       <-chan collectorRunResult
		retryTimer    *time.Timer
		retry         <-chan time.Time
		stopRequested bool
	)

	stopRetry := func() {
		if retryTimer != nil && !retryTimer.Stop() {
			select {
			case <-retryTimer.C:
			default:
			}
		}

		retryTimer = nil
		retry = nil
	}

	start := func() {
		runCtx, cancel := context.WithCancel(ctx)
		done := make(chan collectorRunResult, 1)

		runCancel = cancel
		runDone = done
		stopRequested = false
		s.setRunning(true)
		s.logger.Info("Starting collector", "subscribers", s.demand.Subscribers())

		go func() {
			started := s.now()
			err := s.run(runCtx)
			done <- collectorRunResult{err: err, duration: s.now().Sub(started)}
		}()
	}

	if s.demand.Subscribers() > 0 {
		start()
	}

	for {
		select {
		case <-ctx.Done():
			stopRetry()

			if runCancel != nil {
				runCancel()
				<-runDone
				s.setRunning(false)
			}

			s.logger.Info("Collector supervisor stopped")
			return nil

		case <-changes:
			if s.demand.Subscribers() == 0 {
				stopRetry()
				backoff = s.initialBackoff

				if runCancel != nil && !stopRequested {
					stopRequested = true
					s.logger.Info("No subscribers, stopping collector")
					runCancel()
				}

				continue
			}

			// If cancellation is still in flight, its completion path starts the
			// replacement immediately. Starting here would overlap collectors.
			if runDone == nil && retry == nil {
				start()
			}

		case result := <-runDone:
			runCancel = nil
			runDone = nil
			s.setRunning(false)

			if ctx.Err() != nil {
				s.logger.Info("Collector stopped during sidecar shutdown")
				return nil
			}

			wanted := s.demand.Subscribers() > 0
			if !wanted {
				stopRequested = false
				backoff = s.initialBackoff
				s.logger.Info("Collector stopped")
				continue
			}

			if stopRequested {
				// Demand returned while an intentional stop was in flight. Zero
				// demand reset backoff, so reconnect without an artificial delay.
				backoff = s.initialBackoff
				start()
				continue
			}

			if result.duration >= s.backoffReset {
				backoff = s.initialBackoff
			}

			delay := backoff
			backoff = growBackoff(backoff, s.maxBackoff)

			if result.err != nil && !errors.Is(result.err, context.Canceled) {
				s.logger.Error("Collector exited; scheduling restart", "error", result.err, "retry_in", delay)
			} else {
				s.logger.Warn("Collector exited; scheduling restart", "retry_in", delay)
			}

			retryTimer = time.NewTimer(delay)
			retry = retryTimer.C

		case <-retry:
			retryTimer = nil
			retry = nil

			if ctx.Err() != nil {
				continue
			}

			if s.demand.Subscribers() > 0 {
				start()
			} else {
				backoff = s.initialBackoff
			}
		}
	}
}

func growBackoff(current, maximum time.Duration) time.Duration {
	if current >= maximum || current > maximum/2 {
		return maximum
	}

	return current * 2
}
