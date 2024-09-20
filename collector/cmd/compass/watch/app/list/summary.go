package list

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"

	"github.com/skpr/compass/collector/cmd/compass/watch/app/profile"
	"github.com/skpr/compass/collector/pkg/color"
)

// Returns a formatted string of the profile summary.
func profileSummary(profile profile.Profile, active bool) string {
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
