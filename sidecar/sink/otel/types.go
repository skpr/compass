package otel

// Trace collected by the collector.
type Trace struct {
	ResourceSpans []ResourceSpan `json:"resourceSpans"`
}

// ResourceSpan captures information about the entity for which telemetry is recorded.
type ResourceSpan struct {
	Resource   Resource    `json:"resource"`
	ScopeSpans []ScopeSpan `json:"scopeSpans"`
}

// Resource captures information about the entity for which telemetry is recorded.
type Resource struct {
	Attributes []Attribute `json:"attributes"`
}

// Attribute are a list of key/value pairs that are used to identify the type of trace.
type Attribute struct {
	Key   string         `json:"key"`
	Value AttributeValue `json:"value"`
}

// AttributeValue is part of the attribute key/value pair..
type AttributeValue struct {
	StringValue string `json:"stringValue"`
}

// ScopeSpan represents an operation within a transaction.
type ScopeSpan struct {
	Spans []Span `json:"spans"`
}

// Span represents an operation within a transaction.
type Span struct {
	TraceID           string `json:"traceId"`
	SpanID            string `json:"spanId"`
	Name              string `json:"name"`
	Kind              string `json:"kind"` // @todo, Const?
	StartTimeUnixNano int64  `json:"startTimeUnixNano"`
	EndTimeUnixNano   int64  `json:"endTimeUnixNano"`
}
