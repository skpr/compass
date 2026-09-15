// Package spans aggregates the function calls of a request as their events arrive.
package spans

import (
	"bytes"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/skpr/compass/pkg/trace"
)

// DefaultMax is how many spans a trace carries when no limit is configured.
//
// A request calls a few hundred distinct functions, and this bounds the
// product of those and the slices of the request they were called in, so it is
// well clear of what an ordinary request produces while still bounding a
// pathological one.
const DefaultMax = 10_000

// DefaultBucket is how finely calls are placed in time.
//
// Calls of the same function within one bucket become one span, so this is the
// resolution at which a reader can see when a function ran, and it is what
// decides how many spans a request produces:
//
//	spans ≈ distinct functions × request duration / bucket
//
// Ten milliseconds is one percent of a one second request, which is the
// resolution the terminal draws at, and it keeps a request calling a few
// hundred distinct functions for a second inside DefaultMax. A request which
// runs for much longer than that, or calls far more distinct functions, needs
// a coarser bucket or a higher bound — otherwise the calls which find no span
// are counted rather than placed. See docs/scaling.md.
//
// It is absolute rather than a share of the request because a span has to be
// chosen when its call arrives, and how long the request ran for is not known
// until it ends.
const DefaultBucket = 10 * time.Millisecond

// MaxNames bounds the names one aggregator holds on to.
//
// A request calls a few hundred distinct functions and an application has a
// few thousand, so this is far above what a real workload reaches. It is here
// because the names come from the application rather than from us: code which
// generates function names at runtime would otherwise grow this map for as
// long as the collector runs. Past the bound the names are still reported,
// they are just no longer reused.
const MaxNames = 8192

// Runtime is a fixed metric label for the runtime which dropped a call.
type Runtime string

// Runtime metric labels.
const (
	RuntimeNodeHTTP Runtime = "node_http"
	RuntimePHPCLI   Runtime = "php_cli"
	RuntimePHPFPM   Runtime = "php_fpm"
)

var metricCallsDropped = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "compass_sidecar_function_events_dropped_total",
	Help: "The total number of function events no span represents, because a trace reached its configured limit.",
}, []string{"runtime"})

// Options bound and shape the aggregation. The zero value uses the defaults
// above.
type Options struct {
	// Max spans a trace may carry.
	Max int
	// Bucket is how finely calls are placed in time.
	Bucket time.Duration
}

// Aggregator is one runtime's aggregation policy, and the function names seen
// so far.
//
// It lives for as long as the collector does and is shared by the requests it
// builds, which is what lets a name be turned into a string once rather than
// once per call.
type Aggregator struct {
	max     int
	bucket  time.Duration
	runtime Runtime
	names   map[string]string
}

// New creates an aggregator.
func New(options Options, runtime Runtime) *Aggregator {
	maxSpans := options.Max
	if maxSpans <= 0 {
		maxSpans = DefaultMax
	}

	bucket := options.Bucket
	if bucket <= 0 {
		bucket = DefaultBucket
	}

	switch runtime {
	case RuntimeNodeHTTP, RuntimePHPCLI, RuntimePHPFPM:
	default:
		panic(fmt.Sprintf("unknown span runtime %q", runtime))
	}

	return &Aggregator{
		max:     maxSpans,
		bucket:  bucket,
		runtime: runtime,
		names:   make(map[string]string, MaxNames/8),
	}
}

// Max spans a trace built by this aggregator may carry.
func (a *Aggregator) Max() int { return a.max }

// Bucket is how finely this aggregator places calls in time.
func (a *Aggregator) Bucket() time.Duration { return a.bucket }

// Builder accumulates the calls of a single request.
//
// It is not safe for concurrent use: the handler which owns the request holds
// a lock across the whole of an event anyway.
type Builder struct {
	aggregator *Aggregator
	index      map[key]int
	spans      []trace.Span
	calls      int64
	dropped    int64
	maxMemory  int64
}

// key identifies the span a call is aggregated into: the function, and the
// slice of the request it ran in.
type key struct {
	name   string
	bucket int64
}

// Request starts a builder for one request.
func (a *Aggregator) Request() *Builder {
	return &Builder{
		aggregator: a,
		index:      make(map[key]int),
	}
}

// Add one function call to the request being built.
//
// Every call is counted and every call's memory is taken, including those
// which arrive once the trace is full: how much of a request a trace failed to
// represent is itself worth reporting, and peak memory is not a sample.
func (b *Builder) Add(name []byte, offset, elapsed time.Duration, memory int64) {
	b.calls++

	if b.maxMemory < memory {
		b.maxMemory = memory
	}

	// The name is interned first so that the key below holds a string which
	// has already been made, rather than making one per call.
	spanKey := key{
		name:   b.aggregator.name(name),
		bucket: int64(offset / b.aggregator.bucket),
	}

	if index, ok := b.index[spanKey]; ok {
		span := &b.spans[index]

		span.Calls++
		span.Total += elapsed

		if span.Elapsed < elapsed {
			span.Elapsed = elapsed
		}

		if span.Offset > offset {
			span.Offset = offset
		}

		if span.Memory < memory {
			span.Memory = memory
		}

		return
	}

	if len(b.spans) >= b.aggregator.max {
		b.dropped++
		metricCallsDropped.WithLabelValues(string(b.aggregator.runtime)).Inc()

		return
	}

	b.index[spanKey] = len(b.spans)

	b.spans = append(b.spans, trace.Span{
		Name:    spanKey.name,
		Offset:  offset,
		Elapsed: elapsed,
		Total:   elapsed,
		Calls:   1,
		Memory:  memory,
	})
}

// Finish writes what was built into the trace.
func (b *Builder) Finish(tr *trace.Trace) {
	tr.Spans = b.spans
	tr.Calls = b.calls
	tr.CallsDropped = b.dropped

	if tr.ResourceUtilisation.MaxMemory < b.maxMemory {
		tr.ResourceUtilisation.MaxMemory = b.maxMemory
	}
}

// Spans built so far. Used by tests and by callers which report progress on a
// request which has not ended yet.
func (b *Builder) Spans() []trace.Span { return b.spans }

// Calls added so far, including those no span represents.
func (b *Builder) Calls() int64 { return b.calls }

// name of a function as a string, reusing the string already held for it where
// there is one.
//
// The lookup is written as a map index on a string conversion of the bytes,
// which the compiler performs without allocating. Only a name which has not
// been seen before is allocated.
func (a *Aggregator) name(raw []byte) string {
	// The probe writes a NUL-terminated string into a fixed-size field, so the
	// name is whatever precedes the terminator.
	if index := bytes.IndexByte(raw, 0); index >= 0 {
		raw = raw[:index]
	}

	if interned, ok := a.names[string(raw)]; ok {
		return interned
	}

	name := string(raw)

	if a.names != nil && len(a.names) < MaxNames {
		a.names[name] = name
	}

	return name
}
