// Package drupal for summarising the Drupal cacheability data in a trace.
package drupal

import (
	"sort"

	"github.com/skpr/compass/pkg/trace"
)

// Unmarshal a full trace into a summary of its Drupal cacheability data.
func Unmarshal(fullTrace trace.Trace) Summary {
	summary := Summary{
		EffectiveMaxAge: trace.CacheMaxAgePermanent,
	}

	if fullTrace.Drupal == nil {
		return summary
	}

	summary.Dropped = fullTrace.Drupal.CacheEventsDropped

	if len(fullTrace.Drupal.CacheEvents) == 0 {
		return summary
	}

	summary.Collected = true

	var (
		tags     = make(map[string]struct{})
		contexts = make(map[string]struct{})
		blockers = make(map[string]int64)
	)

	for _, event := range fullTrace.Drupal.CacheEvents {
		summary.EffectiveMaxAge = mergeMaxAge(summary.EffectiveMaxAge, event.MaxAge)

		if event.MaxAge == 0 {
			blockers[event.Caller] += event.Calls
		}

		for _, tag := range event.Tags {
			tags[tag] = struct{}{}
		}

		for _, context := range event.Contexts {
			contexts[context] = struct{}{}
		}
	}

	summary.Uncacheable = summary.EffectiveMaxAge == 0
	summary.Tags = sortedKeys(tags)
	summary.Contexts = sortedKeys(contexts)

	for caller, calls := range blockers {
		summary.Blockers = append(summary.Blockers, Blocker{
			Caller: caller,
			Calls:  calls,
		})
	}

	// Blockers came out of a map, which has no ordering, so the most frequent
	// offender is sorted to the top for display.
	sort.Slice(summary.Blockers, func(i, j int) bool {
		if summary.Blockers[i].Calls != summary.Blockers[j].Calls {
			return summary.Blockers[i].Calls > summary.Blockers[j].Calls
		}

		return summary.Blockers[i].Caller < summary.Blockers[j].Caller
	})

	summary.Events = make([]trace.CacheEvent, len(fullTrace.Drupal.CacheEvents))
	copy(summary.Events, fullTrace.Drupal.CacheEvents)

	// Most restrictive first: the events which shortened the max age are the
	// ones worth reading, and a permanently cacheable event never is.
	sort.Slice(summary.Events, func(i, j int) bool {
		var (
			left  = summary.Events[i]
			right = summary.Events[j]
		)

		if left.MaxAge != right.MaxAge {
			return lessMaxAge(left.MaxAge, right.MaxAge)
		}

		if left.Calls != right.Calls {
			return left.Calls > right.Calls
		}

		if left.Caller != right.Caller {
			return left.Caller < right.Caller
		}

		return left.StartTime < right.StartTime
	})

	return summary
}

// mergeMaxAge combines two max ages the way Drupal does, by taking the lowest
// of the two. Permanent is the absence of a limit rather than a low value, so
// anything else wins against it.
func mergeMaxAge(current, next int64) int64 {
	if current == trace.CacheMaxAgePermanent {
		return next
	}

	if next == trace.CacheMaxAgePermanent {
		return current
	}

	if next < current {
		return next
	}

	return current
}

// lessMaxAge orders max ages from most to least restrictive, with permanent
// sorting last.
func lessMaxAge(left, right int64) bool {
	if left == trace.CacheMaxAgePermanent {
		return false
	}

	if right == trace.CacheMaxAgePermanent {
		return true
	}

	return left < right
}

// sortedKeys of a set, so that the output does not depend on map ordering.
func sortedKeys(set map[string]struct{}) []string {
	if len(set) == 0 {
		return nil
	}

	keys := make([]string, 0, len(set))

	for key := range set {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	return keys
}
