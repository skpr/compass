package app

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skpr/compass/pkg/app/events"
)

func TestHistory_AppendIsBoundedAndNewestFirst(t *testing.T) {
	h := newHistory[int](3)

	for value := 1; value <= 5; value++ {
		h.append(value)
	}

	require.Equal(t, 3, h.len())
	for index, want := range []int{5, 4, 3} {
		got, ok := h.newest(index)
		require.True(t, ok)
		assert.Equal(t, want, got)
	}
	for index, want := range []int{3, 4, 5} {
		got, ok := h.oldest(index)
		require.True(t, ok)
		assert.Equal(t, want, got)
	}
}

func TestHistory_AppendAllocationsDoNotScaleWithCapacity(t *testing.T) {
	allocations := func(capacity int) float64 {
		h := newHistory[int](capacity)
		for i := range capacity {
			h.append(i)
		}

		return testing.AllocsPerRun(1000, func() {
			h.append(1)
		})
	}

	small := allocations(100)
	large := allocations(10_000)
	assert.LessOrEqual(t, large, small, "full-ring insertion must not copy retained values")
}

func BenchmarkHistoryAppendAtCapacity(b *testing.B) {
	for _, capacity := range []int{100, 500, 1_000, 10_000} {
		b.Run(fmt.Sprintf("capacity-%d", capacity), func(b *testing.B) {
			h := newHistory[events.Trace](capacity)
			for i := range capacity {
				h.append(newTrace(fmt.Sprintf("trace-%d", i)))
			}
			event := newTrace("next")

			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				h.append(event)
			}
		})
	}
}

func BenchmarkTraceArrivalBurst(b *testing.B) {
	for _, filtered := range []bool{false, true} {
		name := "unfiltered"
		if filtered {
			name = "filtered"
		}

		b.Run(name, func(b *testing.B) {
			const capacity = 500
			m := NewModel("", capacity, capacity)
			m.Init()
			for i := range capacity {
				m.updateTrace(newTrace(fmt.Sprintf("trace-%d", i)))
			}
			if filtered {
				m.filter.SetValue("matching")
				m.searchSetRows()
			}
			event := newTrace("matching-next")

			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				m.updateTrace(event)
			}
		})
	}
}

func BenchmarkLogArrivalBurst(b *testing.B) {
	for _, filtered := range []bool{false, true} {
		name := "unfiltered"
		if filtered {
			name = "filtered"
		}

		b.Run(name, func(b *testing.B) {
			const capacity = 1_000
			m := NewModel("", capacity, capacity)
			m.Init()
			for i := range capacity {
				m.updateLog(testLog(i, "error", fmt.Sprintf("message-%d", i)))
			}
			if filtered {
				m.filter.SetValue("matching")
				m.logsSetRows()
			}
			event := testLog(capacity+1, "error", "matching-next")

			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				m.updateLog(event)
			}
		})
	}
}

func BenchmarkTraceArrivalAtCapacity(b *testing.B) {
	for _, capacity := range []int{100, 500, 1_000, 10_000} {
		b.Run(fmt.Sprintf("capacity-%d", capacity), func(b *testing.B) {
			m := NewModel("", capacity, capacity)
			m.Init()
			for i := range capacity {
				m.updateTrace(newTrace(fmt.Sprintf("trace-%d", i)))
			}
			event := newTrace("next")

			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				m.updateTrace(event)
			}
		})
	}
}

func BenchmarkLogArrivalAtCapacity(b *testing.B) {
	for _, capacity := range []int{100, 500, 1_000, 10_000} {
		b.Run(fmt.Sprintf("capacity-%d", capacity), func(b *testing.B) {
			m := NewModel("", capacity, capacity)
			m.Init()
			for i := range capacity {
				m.updateLog(testLog(i, "info", fmt.Sprintf("message-%d", i)))
			}
			events := []events.Log{
				testLog(capacity+1, "info", "alternating-a"),
				testLog(capacity+2, "info", "alternating-b"),
			}

			b.ReportAllocs()
			b.ResetTimer()
			for i := range b.N {
				m.updateLog(events[i%len(events)])
			}
		})
	}
}
