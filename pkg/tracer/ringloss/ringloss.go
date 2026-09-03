// Package ringloss exports kernel ring-buffer reserve failures as process-level metrics.
package ringloss

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/skpr/compass/pkg/tracer/ingest"
	"github.com/skpr/compass/pkg/tracer/ringreader"
)

const pollInterval = time.Second

const (
	keyEvents uint32 = iota
	keyDrupalCache
)

var metricReserveFailures = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "compass_sidecar_ringbuf_reserve_failures_total",
	Help: "The total number of events rejected because a kernel ring buffer had no space.",
}, []string{"runtime", "stream"})

// CounterMap is the BPF map operation used by Observer.
type CounterMap interface {
	Lookup(key, valueOut interface{}) error
}

type streamState struct {
	key    uint32
	stream ringreader.Stream
	last   uint64
	warned bool
}

// Observer adds one BPF object's counter deltas to process-global metrics.
type Observer struct {
	counterMap CounterMap
	runtime    ingest.Runtime
	streams    []streamState
	counters   *prometheus.CounterVec
	warn       func(ingest.Runtime, ringreader.Stream, uint64)
}

// NewObserver creates an observer for one runtime's BPF objects.
func NewObserver(counterMap CounterMap, runtime ingest.Runtime, streams ...ringreader.Stream) (*Observer, error) {
	return newObserver(counterMap, runtime, metricReserveFailures, defaultWarning, streams...)
}

func newObserver(
	counterMap CounterMap,
	runtime ingest.Runtime,
	counters *prometheus.CounterVec,
	warn func(ingest.Runtime, ringreader.Stream, uint64),
	streams ...ringreader.Stream,
) (*Observer, error) {
	if counterMap == nil {
		return nil, fmt.Errorf("ring-buffer reserve-failure map is required")
	}
	if len(streams) == 0 {
		return nil, fmt.Errorf("at least one ring-buffer stream is required")
	}

	observer := &Observer{
		counterMap: counterMap,
		runtime:    runtime,
		counters:   counters,
		warn:       warn,
		streams:    make([]streamState, 0, len(streams)),
	}
	seen := make(map[ringreader.Stream]struct{}, len(streams))

	for _, stream := range streams {
		if _, ok := seen[stream]; ok {
			return nil, fmt.Errorf("duplicate ring-buffer stream %q", stream)
		}
		seen[stream] = struct{}{}

		key, err := streamKey(runtime, stream)
		if err != nil {
			return nil, err
		}
		observer.streams = append(observer.streams, streamState{key: key, stream: stream})
	}

	return observer, nil
}

// Run periodically exports new failures until collection stops.
func (o *Observer) Run(ctx context.Context) error {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := o.Observe(); err != nil {
				return err
			}
		}
	}
}

// Observe reads every configured stream and exports only its delta since the
// prior read of this BPF object.
func (o *Observer) Observe() error {
	for i := range o.streams {
		state := &o.streams[i]
		var perCPU []uint64
		if err := o.counterMap.Lookup(state.key, &perCPU); err != nil {
			return fmt.Errorf("read ring-buffer reserve failures (runtime=%s, stream=%s): %w", o.runtime, state.stream, err)
		}

		var total uint64
		for _, value := range perCPU {
			total += value
		}
		if total < state.last {
			return fmt.Errorf("ring-buffer reserve-failure counter decreased (runtime=%s, stream=%s, previous=%d, current=%d)", o.runtime, state.stream, state.last, total)
		}

		delta := total - state.last
		state.last = total
		if delta == 0 {
			continue
		}

		o.counters.WithLabelValues(string(o.runtime), string(state.stream)).Add(float64(delta))
		if !state.warned {
			o.warn(o.runtime, state.stream, delta)
			state.warned = true
		}
	}

	return nil
}

// Total returns the BPF map total observed for one stream in this run.
func (o *Observer) Total(stream ringreader.Stream) uint64 {
	for _, state := range o.streams {
		if state.stream == stream {
			return state.last
		}
	}
	return 0
}

func streamKey(runtime ingest.Runtime, stream ringreader.Stream) (uint32, error) {
	switch runtime {
	case ingest.RuntimeNodeHTTP, ingest.RuntimePHPCLI:
		if stream == ringreader.StreamEvents {
			return keyEvents, nil
		}
	case ingest.RuntimePHPFPM:
		switch stream {
		case ringreader.StreamEvents:
			return keyEvents, nil
		case ringreader.StreamDrupalCache:
			return keyDrupalCache, nil
		}
	}
	return 0, fmt.Errorf("unknown ring-buffer stream %q for runtime %q", stream, runtime)
}

func defaultWarning(runtime ingest.Runtime, stream ringreader.Stream, failures uint64) {
	slog.Warn("Kernel ring-buffer reserve failures observed",
		"runtime", runtime,
		"stream", stream,
		"failures", failures,
	)
}
