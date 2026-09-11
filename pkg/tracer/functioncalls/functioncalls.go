// Package functioncalls bounds the function records retained while assembling a trace.
package functioncalls

import (
	"bytes"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/skpr/compass/pkg/trace"
)

// DefaultMax is the maximum number of function calls retained per trace when
// no limit is configured. At the probe's maximum field sizes this leaves ample
// room below the CLI's supported transport line size.
const DefaultMax = 10_000

// MaxNames bounds the names one limiter holds on to.
//
// A request calls a few hundred distinct functions and an application has a
// few thousand, so this is far above what a real workload reaches. It is here
// because the names come from the application rather than from us: code which
// generates function names at runtime would otherwise grow this map for as
// long as the collector runs. Past the bound the names are still reported,
// they are just no longer reused.
const MaxNames = 8192

// Runtime is a fixed metric label for the runtime which dropped an event.
type Runtime string

// Runtime metric labels.
const (
	RuntimeNodeHTTP Runtime = "node_http"
	RuntimePHPCLI   Runtime = "php_cli"
	RuntimePHPFPM   Runtime = "php_fpm"
)

var metricEventsDropped = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "compass_sidecar_function_events_dropped_total",
	Help: "The total number of function events not retained because a trace reached its configured limit.",
}, []string{"runtime"})

// Limiter applies one runtime's retention policy.
//
// The names of the functions it retains are held so that the same function
// called twice is one string rather than two. A Drupal request calls a few
// hundred distinct functions, most of them many times over, so nearly every
// call after the first of its name costs nothing to name.
//
// The map is shared by copies of a Limiter, as maps are, and a limiter which
// was not made by NewLimiter simply does not reuse names.
type Limiter struct {
	max     int
	runtime Runtime
	names   map[string]string
}

// NewLimiter creates a limiter. Non-positive limits use DefaultMax.
func NewLimiter(maxCalls int, runtime Runtime) Limiter {
	if maxCalls <= 0 {
		maxCalls = DefaultMax
	}

	switch runtime {
	case RuntimeNodeHTTP, RuntimePHPCLI, RuntimePHPFPM:
	default:
		panic(fmt.Sprintf("unknown function-call runtime %q", runtime))
	}

	return Limiter{
		max:     maxCalls,
		runtime: runtime,
		names:   make(map[string]string),
	}
}

// Add retains a function call when the trace remains below its limit. Peak
// memory and the exact dropped count are updated even after the cap. The name
// is turned into a Go string only for retained calls, and only the first time
// that name is seen.
func (l Limiter) Add(tr *trace.Trace, name []byte, offset, elapsed time.Duration, memory int64) bool {
	if tr.ResourceUtilisation.MaxMemory < memory {
		tr.ResourceUtilisation.MaxMemory = memory
	}

	if len(tr.FunctionCalls) >= l.max {
		tr.FunctionCallsDropped++
		metricEventsDropped.WithLabelValues(string(l.runtime)).Inc()

		return false
	}

	tr.FunctionCalls = append(tr.FunctionCalls, trace.FunctionCall{
		Name:    l.name(name),
		Offset:  offset,
		Elapsed: elapsed,
		Memory:  memory,
	})

	return true
}

// name of a function as a string, reusing the string already held for it where
// there is one.
//
// The lookup is written as a map index on a string conversion of the bytes,
// which the compiler performs without allocating. Only a name which has not
// been seen before is allocated.
func (l Limiter) name(raw []byte) string {
	// The probe writes a NUL-terminated string into a fixed-size field, so the
	// name is whatever precedes the terminator.
	if index := bytes.IndexByte(raw, 0); index >= 0 {
		raw = raw[:index]
	}

	if interned, ok := l.names[string(raw)]; ok {
		return interned
	}

	name := string(raw)

	if l.names != nil && len(l.names) < MaxNames {
		l.names[name] = name
	}

	return name
}
