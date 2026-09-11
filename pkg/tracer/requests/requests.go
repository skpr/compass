// Package requests holds the requests a tracer is still assembling.
package requests

import (
	"time"
)

// DefaultExpire is how long a request may go without an event before it is
// treated as abandoned, when no expiry has been configured.
const DefaultExpire = time.Minute

// Store keeps the state of the requests a tracer has seen the start of but not
// yet the end of, and drops the ones which stop producing events.
//
// A request which never reaches its shutdown probe — the process was killed,
// the probe was missed, the collector attached mid-request — would otherwise
// stay here for the life of the collector. Expiry is measured from the last
// event a request produced rather than from its start, so a long-running
// Drush command is kept for as long as it keeps calling functions.
//
// # Concurrency
//
// A Store is not safe for concurrent use. Its callers hold a lock across the
// read-modify-write of a request's state regardless, because the value handed
// back is a pointer into the store, so a lock in here would be a second one on
// the hottest path in the process — which is what this type exists to remove.
//
// # Time
//
// Every method takes the monotonic timestamp of the event being handled,
// which is the clock the probes report against. Expiry therefore advances with
// the events rather than with the wall clock: nothing here reads the system
// clock, and a store which is receiving no events is not growing either.
type Store[K comparable, V any] struct {
	expire    uint64
	entries   map[K]*entry[V]
	lastSweep uint64
	// swept is whether lastSweep has been set, which a zero timestamp cannot
	// say on its own. Probe timestamps are never zero, but a replayed event or
	// a test starting from zero would otherwise leave the store never sweeping.
	swept bool
}

type entry[V any] struct {
	value *V
	// lastSeen is when this request last produced an event.
	lastSeen uint64
}

// New creates a store which drops requests that have been silent for expire.
// Non-positive values use DefaultExpire.
func New[K comparable, V any](expire time.Duration) *Store[K, V] {
	if expire <= 0 {
		expire = DefaultExpire
	}

	return &Store[K, V]{
		expire:  uint64(expire.Nanoseconds()),
		entries: make(map[K]*entry[V]),
	}
}

// Set records the state of a request which has just started, and is the point
// at which the store sweeps whatever has since been abandoned.
//
// Sweeping here rather than on a ticker keeps the store free of a goroutine
// and a shutdown path, and puts the walk where the growth comes from: the map
// only gains entries through this method.
func (s *Store[K, V]) Set(key K, value *V, now uint64) {
	s.sweep(now)

	s.entries[key] = &entry[V]{value: value, lastSeen: now}
}

// Get the state of a request which is still being assembled, and record that
// it is still alive.
//
// The value is a pointer into the store, so the caller has to hold its lock
// for as long as it uses it. Keeping the request alive costs a field write
// here: pushing an expiry out through a cache, which is what this replaces,
// cost a second map write and a second lock for every function call.
func (s *Store[K, V]) Get(key K, now uint64) (*V, bool) {
	found, ok := s.entries[key]
	if !ok {
		return nil, false
	}

	found.lastSeen = now

	return found.value, true
}

// Delete the request, which the caller does once it has completed one.
func (s *Store[K, V]) Delete(key K) {
	delete(s.entries, key)
}

// Len is how many requests are being assembled. Used by tests and metrics.
func (s *Store[K, V]) Len() int {
	return len(s.entries)
}

// sweep drops every request which has been silent for longer than the expiry,
// no more than once per expiry period.
func (s *Store[K, V]) sweep(now uint64) {
	// The first event a store ever sees sets the baseline rather than sweeping
	// an empty map, and a timestamp which goes backwards — which the monotonic
	// clock does not do, but a test or a replayed event might — only resets it.
	if !s.swept || now < s.lastSweep {
		s.swept = true
		s.lastSweep = now

		return
	}

	if now-s.lastSweep < s.expire {
		return
	}

	s.lastSweep = now

	for key, found := range s.entries {
		if now >= found.lastSeen && now-found.lastSeen >= s.expire {
			delete(s.entries, key)
		}
	}
}
