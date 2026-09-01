// Package functioncalls bounds the function records retained while assembling a trace.
package functioncalls

import (
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"golang.org/x/sys/unix"

	"github.com/skpr/compass/pkg/trace"
)

// DefaultMax is the maximum number of function calls retained per trace when
// no limit is configured. At the probe's maximum field sizes this leaves ample
// room below the CLI's supported transport line size.
const DefaultMax = 10_000

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
type Limiter struct {
	max     int
	runtime Runtime
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

	return Limiter{max: maxCalls, runtime: runtime}
}

// Add retains a function call when the trace remains below its limit. Peak
// memory and the exact dropped count are updated even after the cap. The name
// is converted to a Go string only for retained calls.
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
		Name:    unix.ByteSliceToString(name),
		Offset:  offset,
		Elapsed: elapsed,
		Memory:  memory,
	})

	return true
}
