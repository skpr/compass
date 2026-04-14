package events

// SummaryResult is sent when an Ollama summary request completes successfully.
type SummaryResult struct {
	TraceID string
	Text    string
}

// SummaryError is sent when an Ollama summary request fails.
type SummaryError struct {
	TraceID string
	Err     error
}
