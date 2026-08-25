// Package trace implements complete tracing data.
package trace

import (
	"time"
)

// Source describes the invocation mechanism of the trace.
type Source string

var (
	// SourceHTTP is the source for traces collected from HTTP requests.
	SourceHTTP Source = "http"
	// SourceCLI is the source for traces collected from the command line interface.
	SourceCLI Source = "cli"
)

// Runtime describes the language runtime that produced the trace.
type Runtime string

var (
	// RuntimePHP is the runtime for traces collected from PHP processes.
	RuntimePHP Runtime = "php"
	// RuntimeNode is the runtime for traces collected from Node.js processes.
	RuntimeNode Runtime = "node"
)

// IDUnknown is what the extension sends when a request carried no
// HTTP_X_REQUEST_ID header, and there was therefore nothing to identify it by.
//
// It is a sentinel rather than an identifier: two traces reading UNKNOWN are
// not the same request, so anything correlating on the ID has to treat it as
// absent.
const IDUnknown = "UNKNOWN"

// Metadata associated with this trace.
type Metadata struct {
	Source  Source       `json:"source"`
	Runtime Runtime      `json:"runtime"`
	ID      string       `json:"id"`
	HTTP    MetadataHTTP `json:"http,omitempty"`
	CLI     MetadataCLI  `json:"cli,omitempty"`
	// StartTime and EndTime of the request, as instants.
	//
	// These two are what a reader outside this process correlates against: its
	// own logs, another trace, a deployment. Everything inside the trace is
	// placed relative to StartTime instead.
	StartTime time.Time `json:"startTime"`
	EndTime   time.Time `json:"endTime"`
}

// ExecutionTime of the trace.
func (m Metadata) ExecutionTime() time.Duration {
	return m.EndTime.Sub(m.StartTime)
}

// Identified reports whether the trace carries an ID worth showing.
func (m Metadata) Identified() bool {
	return m.ID != "" && m.ID != IDUnknown
}

type MetadataHTTP struct {
	Method string `json:"method,omitempty"`
	URI    string `json:"uri,omitempty"`
}

type MetadataCLI struct {
	Command string `json:"command,omitempty"`
}

// Trace data collected for a request.
type Trace struct {
	Metadata            Metadata            `json:"metadata"`
	ResourceUtilisation ResourceUtilisation `json:"resourceUtilisation"`
	FunctionCalls       []FunctionCall      `json:"functionCalls"`
	Drupal              *Drupal             `json:"drupal,omitempty"`
}

type ResourceUtilisation struct {
	MaxMemory int64 `json:"maxMemory"`
}

// FunctionCall provides information about the function call.
//
// The call is placed by how far into the request it started rather than by the
// instant it started at. An absolute time for one frame says nothing on its
// own — every consumer immediately subtracts the request start to get back to
// this — and it costs a formatted timestamp per call on the wire.
type FunctionCall struct {
	Name string `json:"name"`
	// Offset from the start of the request at which the call began.
	Offset time.Duration `json:"offsetNanos"`
	// Elapsed is how long the call ran for.
	Elapsed time.Duration `json:"elapsedNanos"`
	Memory  int64         `json:"memory"`
}

// Drupal specific data collected for a trace.
//
// Only populated for PHP traces where the application called into the Drupal
// APIs which Compass probes, so it is absent for Node traces, PHP CLI traces
// and non-Drupal PHP applications.
type Drupal struct {
	// CacheEvents which occurred during this trace, aggregated by their contents.
	CacheEvents []CacheEvent `json:"cacheEvents,omitempty"`
	// CacheEventsDropped is how many distinct cache events were discarded because
	// the trace reached its retention limit. Reported so that a truncated trace is
	// not mistaken for a complete one.
	CacheEventsDropped int `json:"cacheEventsDropped,omitempty"`
}

// CacheOrigin describes which Drupal API produced a cache event.
type CacheOrigin string

var (
	// CacheOriginRenderArray is for cacheability collected from a render array.
	CacheOriginRenderArray CacheOrigin = "renderArray"
	// CacheOriginObject is for cacheability collected from an object.
	CacheOriginObject CacheOrigin = "object"
)

// CacheMaxAgePermanent is the max age Drupal uses for "cacheable forever".
const CacheMaxAgePermanent int64 = -1

// CacheEvent provides the cacheability metadata that Drupal derived at a point
// in the request.
//
// Drupal derives cacheability far more often than a request has interesting
// function calls, so identical events are aggregated into a single entry with a
// call count rather than retained individually.
type CacheEvent struct {
	// Origin of the cacheability metadata.
	Origin CacheOrigin `json:"origin"`
	// Caller which asked Drupal for the cacheability metadata.
	Caller string `json:"caller"`
	// ObjectType that the metadata was derived from. Only set for CacheOriginObject.
	ObjectType string `json:"objectType,omitempty"`
	// MaxAge in seconds, where CacheMaxAgePermanent means cacheable forever.
	MaxAge int64 `json:"maxAge"`
	// Tags which the cached item is invalidated by.
	Tags []string `json:"tags,omitempty"`
	// Contexts which the cached item varies by.
	Contexts []string `json:"contexts,omitempty"`
	// Offset from the start of the request at which the first of these events
	// occurred.
	Offset time.Duration `json:"offsetNanos"`
	// Calls is how many identical events were aggregated into this one.
	Calls int64 `json:"calls"`
}
