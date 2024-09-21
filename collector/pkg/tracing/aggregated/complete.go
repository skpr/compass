// Package aggregated for streamlined profiling data.
package aggregated

import "github.com/skpr/compass/collector/pkg/tracing/complete"

// FromCompleteProfile will convert a complete profile to an aggregated profile.
func FromCompleteProfile(upstream complete.Profile) Profile {
	profile := Profile{
		RequestID:     upstream.RequestID,
		IngestedTime:  upstream.IngestedTime,
		ExecutionTime: upstream.ExecutionTime,
		Functions:     make(map[string]Function),
	}

	for _, call := range upstream.FunctionCalls {
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
