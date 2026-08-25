package segmented

import (
	"fmt"

	"github.com/skpr/compass/pkg/trace"
)

// Trace data collected for a request.
type Trace struct {
	// Metadata associated with this trace.
	Metadata trace.Metadata `json:"metadata"`
	// Total number of segments in this trace.
	Segments int64 `json:"segments"`
	// TotalFunctionCalls that occurred during this trace.
	TotalFunctionCalls int `json:"totalFunctionCalls"`
	// Spans that are included in trace.
	Spans []Span `json:"spans"`
}

// Span provides information about a function call during a .
type Span struct {
	// Name of the function.
	Name string `json:"name"`
	// The original start time of the function called in the span.
	StartTime int64 `json:"startTime"`
	// Which segment this function started.
	Start int64 `json:"start"`
	// How many segments this function call spans.
	Length int64 `json:"length"`
	// TotalFunctionCalls that were called during this span.
	TotalFunctionCalls int `json:"calls"`
	// MaxMemory used during this span.
	MaxMemory int64 `json:"maxMemory"`
	// SelfTime is how long the calls in this span spent doing their own work,
	// rather than waiting on the calls they made. See SelfTime.
	SelfTime int64 `json:"selfTime"`
}

// SelfShare of the request this span was itself responsible for.
func (s Span) SelfShare(executionTime int64) float64 {
	if executionTime <= 0 {
		return 0
	}

	return float64(s.SelfTime) / float64(executionTime)
}

// GetName of the span and include the amount when more than one call.
func (s Span) GetName() string {
	if s.TotalFunctionCalls > 1 {
		return fmt.Sprintf("%s (%d)", s.Name, s.TotalFunctionCalls)
	}

	return s.Name
}
