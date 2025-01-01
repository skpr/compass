package description

// View renders the help component.
func (m Model) View() string {
	if len(m.Trace.FunctionCalls) == 0 {
		return "No tracing data available"
	}

	description, err := getDescription(m.Trace)
	if err != nil {
		return err.Error()
	}

	return description
}
