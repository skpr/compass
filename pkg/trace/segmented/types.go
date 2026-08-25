package segmented

import (
	"fmt"
	"time"

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
	// Offset from the start of the request at which the earliest call in this
	// span began.
	Offset time.Duration `json:"offsetNanos"`
	// Length of time this function call ran for.
	Length time.Duration `json:"lengthNanos"`
	// TotalFunctionCalls that were called during this span.
	TotalFunctionCalls int `json:"calls"`
	// MaxMemory used during this span.
	MaxMemory int64 `json:"maxMemory"`
	// SelfTime is how long the calls in this span spent doing their own work,
	// rather than waiting on the calls they made. See SelfTime.
	SelfTime time.Duration `json:"selfTimeNanos"`
}

// SelfShare of the request this span was itself responsible for.
func (s Span) SelfShare(executionTime time.Duration) float64 {
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
