package app

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"

	"github.com/skpr/compass/collector/pkg/color"
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

// WrapProfile wraps a tracing.Profile into a Profile.
func WrapProfile(profile tracing.Profile) Profile {
	return Profile{
		IngestionTime: time.Now(),
		RequestID:     profile.RequestID,
		ExecutionTime: profile.ExecutionTime,
		Functions:     wrapFunctions(profile.Functions),
	}
}

// wrapFunctions wraps a map of tracing.FunctionSummary into a slice of Function.
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

// Helper function which returns a formatted string of the profile summary.
func profileSummary(profile Profile, active bool) string {
	var totalInvocations int32

	for _, f := range profile.Functions {
		totalInvocations += f.Invocations
	}

	msg := fmt.Sprintf("Request ID = %s\nIngestion Time = %s\nExecution Time = %v\nFunction calls = %d", profile.RequestID, profile.IngestionTime.Format("15:04:05"), profile.ExecutionTime, totalInvocations)

	var style = lipgloss.NewStyle().Padding(0, 1)

	if active {
		return style.Foreground(lipgloss.Color(color.White)).Bold(true).Render(msg)
	}

	return style.Foreground(lipgloss.Color(color.Grey)).Render(msg)
}
