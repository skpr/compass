// Package aggregated for streamlined profiling data.
package aggregated

import (
	"github.com/skpr/compass/profile/complete"
)

// FromCompleteProfile will convert a complete profile to an aggregated profile.
func FromCompleteProfile(upstream complete.Profile) Profile {
	profile := Profile{
		RequestID:     upstream.RequestID,
		StartTime:     upstream.StartTime,
		ExecutionTime: upstream.ExecutionTime,
		Functions:     make(map[string]Function),
	}

	// Consolidate function call that might have occurred int parallel.
	var consolidated []complete.FunctionCall

	for _, call := range upstream.FunctionCalls {
		if !listContains(consolidated, call.Name) {
			consolidated = append(consolidated, call)
			continue
		}

		for i, c := range consolidated {
			if c.Name != call.Name {
				continue
			}

			if call.StartTime > c.StartTime && call.StartTime < c.EndTime {
				consolidated[i].EndTime = c.EndTime
				continue
			}

			if call.EndTime > c.StartTime && call.EndTime < c.EndTime {
				consolidated[i].StartTime = c.StartTime
				continue
			}
		}
	}

	// Use the consolidated function calls to generated our new aggregated list.
	for _, call := range consolidated {
		if _, ok := profile.Functions[call.Name]; !ok {
			profile.Functions[call.Name] = Function{
				Name:          call.Name,
				ExecutionTime: 0,
			}
		}

		executionTime := (call.EndTime - call.StartTime) / 1000

		profile.Functions[call.Name] = Function{
			Name:          call.Name,
			ExecutionTime: profile.Functions[call.Name].ExecutionTime + executionTime,
			Invocations:   profile.Functions[call.Name].Invocations + 1,
		}
	}

	return profile
}

func listContains(list []complete.FunctionCall, name string) bool {
	for _, item := range list {
		if item.Name == name {
			return true
		}
	}

	return false
}
