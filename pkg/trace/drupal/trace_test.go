package drupal

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/skpr/compass/pkg/trace"
)

func TestUnmarshal_NoDrupalData(t *testing.T) {
	summary := Unmarshal(trace.Trace{})

	assert.False(t, summary.Collected)
	assert.Equal(t, trace.CacheMaxAgePermanent, summary.EffectiveMaxAge)
	assert.False(t, summary.Uncacheable)
	assert.Empty(t, summary.Events)
}

func TestUnmarshal_EmptyEventsStillReportsDropped(t *testing.T) {
	summary := Unmarshal(trace.Trace{
		Drupal: &trace.Drupal{
			CacheEventsDropped: 12,
		},
	})

	assert.False(t, summary.Collected)
	assert.Equal(t, 12, summary.Dropped)
}

func TestUnmarshal_EffectiveMaxAge(t *testing.T) {
	tests := []struct {
		name     string
		maxAges  []int64
		expected int64
	}{
		{
			name:     "all permanent",
			maxAges:  []int64{-1, -1, -1},
			expected: -1,
		},
		{
			name:     "permanent and a limit",
			maxAges:  []int64{-1, 3600, -1},
			expected: 3600,
		},
		{
			name:     "lowest limit wins",
			maxAges:  []int64{3600, 60, 900},
			expected: 60,
		},
		{
			name:     "zero beats everything",
			maxAges:  []int64{-1, 3600, 0, 60},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var events []trace.CacheEvent

			for _, maxAge := range tt.maxAges {
				events = append(events, trace.CacheEvent{
					Origin: trace.CacheOriginRenderArray,
					Caller: "Some\\Caller::method",
					MaxAge: maxAge,
					Calls:  1,
				})
			}

			summary := Unmarshal(trace.Trace{
				Drupal: &trace.Drupal{CacheEvents: events},
			})

			assert.True(t, summary.Collected)
			assert.Equal(t, tt.expected, summary.EffectiveMaxAge)
			assert.Equal(t, tt.expected == 0, summary.Uncacheable)
		})
	}
}

func TestUnmarshal_TagsAndContextsAreUnioned(t *testing.T) {
	summary := Unmarshal(trace.Trace{
		Drupal: &trace.Drupal{
			CacheEvents: []trace.CacheEvent{
				{
					Caller:   "A::a",
					MaxAge:   -1,
					Tags:     []string{"node:1", "config:system.site"},
					Contexts: []string{"url.path"},
					Calls:    1,
				},
				{
					Caller:   "B::b",
					MaxAge:   -1,
					Tags:     []string{"node:1", "node:2"},
					Contexts: []string{"user.roles", "url.path"},
					Calls:    1,
				},
			},
		},
	})

	assert.Equal(t, []string{"config:system.site", "node:1", "node:2"}, summary.Tags)
	assert.Equal(t, []string{"url.path", "user.roles"}, summary.Contexts)
}

func TestUnmarshal_Blockers(t *testing.T) {
	summary := Unmarshal(trace.Trace{
		Drupal: &trace.Drupal{
			CacheEvents: []trace.CacheEvent{
				{Caller: "Zebra::render", MaxAge: 0, Calls: 3},
				{Caller: "Apple::render", MaxAge: 0, Calls: 9},
				{Caller: "Apple::render", MaxAge: 0, Calls: 1, Contexts: []string{"session"}},
				{Caller: "Cacheable::render", MaxAge: 3600, Calls: 40},
			},
		},
	})

	assert.True(t, summary.Uncacheable)
	assert.Equal(t, []Blocker{
		{Caller: "Apple::render", Calls: 10},
		{Caller: "Zebra::render", Calls: 3},
	}, summary.Blockers)
}

func TestUnmarshal_EventsOrderedMostRestrictiveFirst(t *testing.T) {
	summary := Unmarshal(trace.Trace{
		Drupal: &trace.Drupal{
			CacheEvents: []trace.CacheEvent{
				{Caller: "permanent", MaxAge: -1, Calls: 100},
				{Caller: "hour", MaxAge: 3600, Calls: 1},
				{Caller: "uncacheable", MaxAge: 0, Calls: 2},
				{Caller: "minute", MaxAge: 60, Calls: 1},
			},
		},
	})

	var callers []string

	for _, event := range summary.Events {
		callers = append(callers, event.Caller)
	}

	assert.Equal(t, []string{"uncacheable", "minute", "hour", "permanent"}, callers)
}

func TestUnmarshal_DoesNotMutateTrace(t *testing.T) {
	full := trace.Trace{
		Drupal: &trace.Drupal{
			CacheEvents: []trace.CacheEvent{
				{Caller: "permanent", MaxAge: -1, Calls: 1},
				{Caller: "uncacheable", MaxAge: 0, Calls: 1},
			},
		},
	}

	Unmarshal(full)

	assert.Equal(t, "permanent", full.Drupal.CacheEvents[0].Caller)
}
