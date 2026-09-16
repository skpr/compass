package http

import (
	"github.com/skpr/compass/pkg/trace"
)

// MaxInternedNames bounds the strings one stream holds on to.
//
// An application has a few thousand distinct functions, and its cache tags and
// contexts are drawn from a similar sized vocabulary, so this is far above what
// a real workload reaches. It is here because the strings come from the
// application rather than from us: code which generates function names at
// runtime, or cache tags carrying an entity ID, would otherwise grow this map
// for as long as the CLI runs. Past the bound the strings are still carried,
// they are just no longer shared.
const MaxInternedNames = 8192

// interner holds one string per distinct name a stream has carried, and hands
// that one back for every later trace which names it again.
//
// A trace is one span per function per slice of the request it ran in, so a
// function live across a one second request arrives as a hundred spans, each
// with its own copy of a name which is often fifty characters. Nothing about
// the decoder knows they are the same string, so a retained trace holds
// thousands of duplicates: about half of what it weighs. Sharing them is what
// lets the byte budget retain roughly twice the traces, and it is shared across
// the whole stream rather than within one trace, because every request through
// an application calls much the same functions as the last one.
//
// It is not safe for concurrent use: the goroutine reading the stream owns it.
type interner struct {
	strings map[string]string
}

// newInterner for one run of the stream client.
func newInterner() *interner {
	return &interner{strings: make(map[string]string, MaxInternedNames/8)}
}

// trace replaces the strings in a decoded trace with the ones already held for
// them, where there are any.
func (i *interner) trace(t *trace.Trace) {
	if i == nil {
		return
	}

	for index := range t.Spans {
		t.Spans[index].Name = i.intern(t.Spans[index].Name)
	}

	if t.Drupal == nil {
		return
	}

	// Drupal derives cacheability from the same callers over and over, and the
	// tags and contexts it reports repeat across the events of a request and
	// across requests, so they are worth sharing for the same reason the
	// function names are.
	for index := range t.Drupal.CacheEvents {
		event := &t.Drupal.CacheEvents[index]

		event.Caller = i.intern(event.Caller)
		event.ObjectType = i.intern(event.ObjectType)

		for tag := range event.Tags {
			event.Tags[tag] = i.intern(event.Tags[tag])
		}

		for context := range event.Contexts {
			event.Contexts[context] = i.intern(event.Contexts[context])
		}
	}
}

// intern one string, returning the copy already held for it where there is one.
func (i *interner) intern(s string) string {
	if s == "" {
		return s
	}

	if held, ok := i.strings[s]; ok {
		return held
	}

	if len(i.strings) < MaxInternedNames {
		i.strings[s] = s
	}

	return s
}
