package aggregated

import "github.com/skpr/compass/collector/pkg/tracing/complete"

// Convert the upstream (full) profile to a stdout acceptable profile.
func FromCompleteProfile(upstream complete.Profile) Profile {
	profile := Profile{
		RequestID:     upstream.RequestID,
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

		profile.Functions[call.Name] = Function{
			Name:          call.Name,
			ExecutionTime: profile.Functions[call.Name].ExecutionTime + call.EndTime - call.StartTime,
			Invocations:   profile.Functions[call.Name].Invocations + 1,
		}
	}

	return profile
}
