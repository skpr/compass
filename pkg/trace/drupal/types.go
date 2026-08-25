package drupal

import (
	"github.com/skpr/compass/pkg/trace"
)

// Summary of the Drupal cacheability data collected during a trace.
type Summary struct {
	// Collected is true when the trace carried at least one cache event. The
	// zero value of the other fields is indistinguishable from "everything was
	// permanently cacheable", so this reports whether they mean anything.
	Collected bool `json:"collected"`
	// EffectiveMaxAge is the max age the response ends up with, which Drupal
	// derives by taking the lowest of the max ages that contributed to it.
	// trace.CacheMaxAgePermanent means nothing shortened it.
	EffectiveMaxAge int64 `json:"effectiveMaxAge"`
	// Uncacheable is true when something set a max age of zero.
	Uncacheable bool `json:"uncacheable"`
	// Tags is the sorted union of every cache tag seen during the trace.
	Tags []string `json:"tags,omitempty"`
	// Contexts is the sorted union of every cache context seen during the trace.
	Contexts []string `json:"contexts,omitempty"`
	// Blockers are the callers which set a max age of zero, which is what makes
	// a response uncacheable.
	Blockers []Blocker `json:"blockers,omitempty"`
	// Events from the trace, ordered most restrictive first.
	Events []trace.CacheEvent `json:"events,omitempty"`
	// Dropped is how many events the tracer discarded for this trace.
	Dropped int `json:"dropped,omitempty"`
}

// Blocker is a caller which made the response uncacheable.
type Blocker struct {
	// Caller which set a max age of zero.
	Caller string `json:"caller"`
	// Calls is how many times it did so.
	Calls int64 `json:"calls"`
}
