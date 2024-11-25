package trace

// Dedupe the number of function calls.
func (t Trace) Dedupe() Trace {
	var calls []FunctionCall

	for _, call := range t.FunctionCalls {
		if !listContains(calls, call.Name) {
			calls = append(calls, call)
			continue
		}

		for i, c := range calls {
			if c.Name != call.Name {
				continue
			}

			if call.StartTime > c.StartTime && call.StartTime < c.EndTime {
				calls[i].EndTime = c.EndTime
				continue
			}

			if call.EndTime > c.StartTime && call.EndTime < c.EndTime {
				calls[i].StartTime = c.StartTime
				continue
			}
		}
	}

	t.FunctionCalls = calls

	return t
}

func listContains(list []FunctionCall, name string) bool {
	for _, item := range list {
		if item.Name == name {
			return true
		}
	}

	return false
}
