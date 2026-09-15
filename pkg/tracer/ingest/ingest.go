// Package ingest provides shared event decoding and skip handling for runtime tracers.
package ingest

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Errors which describe events that are safe to skip without restarting a tracer.
var (
	ErrRequestNotTracked = errors.New("request not tracked")
	ErrInvalidIdentifier = errors.New("invalid event identifier")
	ErrTraceEmpty        = errors.New("trace has no retained data")
)

// Runtime is a fixed metric label for a runtime event reader.
type Runtime string

// Runtime metric labels.
const (
	RuntimeNodeHTTP Runtime = "node_http"
	RuntimePHPCLI   Runtime = "php_cli"
	RuntimePHPFPM   Runtime = "php_fpm"
)

// Reason is a fixed metric label describing why an event was skipped.
type Reason string

// Skip reason metric labels.
const (
	ReasonRequestNotTracked Reason = "request_not_tracked"
	ReasonInvalidIdentifier Reason = "invalid_identifier"
	ReasonTraceEmpty        Reason = "trace_empty"
)

var metricEventsSkipped = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "compass_sidecar_tracer_events_skipped_total",
	Help: "The total number of tracer events skipped because they could not belong to a complete trace.",
}, []string{"runtime", "reason"})

// Skips records safe-to-ignore outcomes for one tracer run.
type Skips struct {
	runtime Runtime
	counts  [3]atomic.Uint64
}

// NewSkips creates skip counters for a runtime reader.
func NewSkips(runtime Runtime) *Skips {
	switch runtime {
	case RuntimeNodeHTTP, RuntimePHPCLI, RuntimePHPFPM:
	default:
		panic(fmt.Sprintf("unknown tracer runtime %q", runtime))
	}

	return &Skips{runtime: runtime}
}

// Record counts err when it is a known skippable outcome. It returns whether
// the reader should continue.
func (s *Skips) Record(err error) bool {
	reason, index, ok := classify(err)
	if !ok {
		return false
	}

	s.counts[index].Add(1)
	metricEventsSkipped.WithLabelValues(string(s.runtime), string(reason)).Inc()

	return true
}

// Count returns the per-run count for a skip reason.
func (s *Skips) Count(reason Reason) uint64 {
	index, ok := reasonIndex(reason)
	if !ok {
		return 0
	}

	return s.counts[index].Load()
}

// Total returns all events skipped during this tracer run.
func (s *Skips) Total() uint64 {
	return s.counts[0].Load() + s.counts[1].Load() + s.counts[2].Load()
}

// CheckSize requires a ring-buffer sample to be exactly the size of the record
// it is about to be read as.
//
// It is what makes a decoder which reads fields at hand-written offsets safe:
// those offsets come from a generated layout, and a sample of the wrong length
// is the sign that the layout is not the one they were written for. Failing
// here reports that, rather than reading a field from the wrong place.
func CheckSize(rawSample []byte, want int) error {
	if len(rawSample) != want {
		return fmt.Errorf("invalid event size: got %d bytes, want %d", len(rawSample), want)
	}

	return nil
}

// DecodeExact decodes one fixed-layout value after requiring its exact generated size.
//
// This walks the struct with reflection, which costs about 1.6µs for the
// compact function record. That is the right trade for the events which arrive
// once or twice per request, and the wrong one for function calls: those are
// decoded at explicit offsets by each runtime instead. See docs/scaling.md.
func DecodeExact[T any](rawSample []byte) (T, error) {
	var event T
	expected := binary.Size(event)
	if expected < 0 {
		return event, fmt.Errorf("unsupported event layout %T", event)
	}
	if err := CheckSize(rawSample, expected); err != nil {
		return event, err
	}
	if err := binary.Read(bytes.NewReader(rawSample), binary.LittleEndian, &event); err != nil {
		return event, err
	}
	return event, nil
}

// Handle passes one decoded event to its handler. Known incomplete-event
// outcomes are counted and treated as successful reads; all other errors remain
// fatal.
func Handle[T any](ctx context.Context, event T, handle func(context.Context, T) error, skips *Skips) error {
	if err := handle(ctx, event); err != nil {
		if skips.Record(err) {
			return nil
		}

		return fmt.Errorf("failed to handle event: %w", err)
	}
	return nil
}

// DecodeAndHandle decodes one ring-buffer sample and passes it to a handler.
// Known incomplete-event outcomes are counted and treated as successful reads;
// malformed samples and all other handler errors remain fatal.
func DecodeAndHandle[T any](ctx context.Context, rawSample []byte, handle func(context.Context, T) error, skips *Skips) error {
	event, err := DecodeExact[T](rawSample)
	if err != nil {
		return fmt.Errorf("failed to read event: %w", err)
	}
	return Handle(ctx, event, handle, skips)
}

func classify(err error) (Reason, int, bool) {
	switch {
	case errors.Is(err, ErrRequestNotTracked):
		return ReasonRequestNotTracked, 0, true
	case errors.Is(err, ErrInvalidIdentifier):
		return ReasonInvalidIdentifier, 1, true
	case errors.Is(err, ErrTraceEmpty):
		return ReasonTraceEmpty, 2, true
	default:
		return "", 0, false
	}
}

func reasonIndex(reason Reason) (int, bool) {
	switch reason {
	case ReasonRequestNotTracked:
		return 0, true
	case ReasonInvalidIdentifier:
		return 1, true
	case ReasonTraceEmpty:
		return 2, true
	default:
		return 0, false
	}
}
