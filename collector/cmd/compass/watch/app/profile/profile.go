package profile

import (
	"time"

	"github.com/charmbracelet/bubbles/list"

	"github.com/skpr/compass/collector/pkg/tracing"
)

// Profile represents a wrapped Compass profile.
type Profile struct {
	list.Item
	IngestionTime time.Time
	RequestID     string
	ExecutionTime float64
	Functions     []Function
}

// Function represents a wrapped Compass function.
type Function struct {
	Name               string
	TotalExecutionTime float64
	Invocations        int32
}

// Wrap a tracing.Profile into a Profile.
func Wrap(profile tracing.Profile) Profile {
	return Profile{
		IngestionTime: time.Now(),
		RequestID:     profile.RequestID,
		ExecutionTime: profile.ExecutionTime,
		Functions:     wrapFunctions(profile.Functions),
	}
}

// Wraps a map of tracing.FunctionSummary into a slice of Function.
func wrapFunctions(functions map[string]tracing.FunctionSummary) []Function {
	var fs []Function

	for name, f := range functions {
		fs = append(fs, Function{
			Name:               name,
			TotalExecutionTime: f.TotalExecutionTime,
			Invocations:        f.Invocations,
		})
	}

	return fs
}
