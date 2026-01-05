// Package count for storing tracing data in counted totals.
package count

import (
	"sort"

	"github.com/skpr/compass/tracing/trace"
	"github.com/skpr/compass/tracing/trace/aggregated"
)

// Unmarshal a full trace into a counted trace.
func Unmarshal(fullTrace trace.Trace) Trace {
	// We first unmarshal it into segments so we can determine the percentage.
	// 100 allows us to compute a percentage.
	segementedTrace := aggregated.Unmarshal(fullTrace)

	functions := make(map[string]Function)

	for _, span := range segementedTrace.Spans {
		function := Function{
			Name:       span.Name,
			Calls:      1,
			Percentage: int64((float64(span.Elapsed) / float64(fullTrace.Metadata.ExecutionTime())) * 100),
		}

		if val, ok := functions[function.Name]; ok {
			function.Calls = val.Calls + function.Calls
			function.Percentage = val.Calls + function.Percentage
			functions[function.Name] = function
			continue
		}

		functions[function.Name] = function
	}

	countedTrace := Trace{
		Metadata:           fullTrace.Metadata,
		TotalFunctionCalls: len(fullTrace.FunctionCalls),
	}

	for _, function := range functions {
		countedTrace.Functions = append(countedTrace.Functions, function)
	}

	// We also need to sort these now that all the spans have gone through a map which does not have ordering.
	sort.Slice(countedTrace.Functions, func(i, j int) bool {
		if countedTrace.Functions[i].Percentage != countedTrace.Functions[j].Percentage {
			return countedTrace.Functions[i].Percentage > countedTrace.Functions[j].Percentage
		}

		if countedTrace.Functions[i].Calls != countedTrace.Functions[j].Calls {
			return countedTrace.Functions[i].Calls > countedTrace.Functions[j].Calls
		}

		return countedTrace.Functions[i].Name < countedTrace.Functions[j].Name
	})

	return countedTrace
}
