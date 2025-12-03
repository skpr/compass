package aggregated

import (
	"fmt"
	"time"

	"github.com/skpr/compass/tracing/trace"
)

// Trace data collected for a request.
type Trace struct {
	// Metadata associated with this trace.
	Metadata trace.Metadata `json:"metadata"`
	// TotalFunctionCalls that occurred during this trace.
	TotalFunctionCalls int `json:"totalFunctionCalls"`
	// Spans that are included in trace.
	Spans []Span `json:"spans"`
}

// Span provides information about a function call during a .
type Span struct {
	// Name of the function.
	Name string `json:"name"`
	// When this function started in the trace.
	Start time.Duration `json:"start"`
	// When this function ended in the trace.
	End time.Duration `json:"end"`
	// How long this function was called for.
	Elapsed time.Duration `json:"elapsed"`
	// Number of times this function was called.
	Calls int `json:"call"`
}

// GetName of the span and include the amount when more than one call.
func (s Span) GetName() string {
	if s.Calls > 1 {
		return fmt.Sprintf("%s (%d)", s.Name, s.Calls)
	}

	return s.Name
}
