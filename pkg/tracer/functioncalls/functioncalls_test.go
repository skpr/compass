package functioncalls

import (
	"fmt"
	"testing"
	"time"
	"unsafe"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skpr/compass/pkg/trace"
)

func TestLimiter_Add_BoundsAndCounts(t *testing.T) {
	limiter := NewLimiter(2, RuntimeNodeHTTP)
	tr := trace.Trace{}
	name := []byte("example\\function\x00")
	counter := metricEventsDropped.WithLabelValues(string(RuntimeNodeHTTP))
	before := testutil.ToFloat64(counter)

	assert.True(t, limiter.Add(&tr, name, time.Nanosecond, 2*time.Nanosecond, 10))
	assert.True(t, limiter.Add(&tr, name, 3*time.Nanosecond, 4*time.Nanosecond, 20))
	assert.False(t, limiter.Add(&tr, name, 5*time.Nanosecond, 6*time.Nanosecond, 30))
	assert.False(t, limiter.Add(&tr, name, 7*time.Nanosecond, 8*time.Nanosecond, 40))

	assert.Len(t, tr.FunctionCalls, 2)
	assert.Equal(t, 2, tr.FunctionCallsDropped)
	assert.Equal(t, int64(40), tr.ResourceUtilisation.MaxMemory)
	assert.Equal(t, before+2, testutil.ToFloat64(counter))
}

func TestLimiter_Add_DefaultLimit(t *testing.T) {
	limiter := NewLimiter(0, RuntimePHPCLI)
	tr := trace.Trace{}
	name := []byte("function\x00")

	for i := 0; i < DefaultMax+1; i++ {
		limiter.Add(&tr, name, 0, 0, 0)
	}

	assert.Len(t, tr.FunctionCalls, DefaultMax)
	assert.Equal(t, 1, tr.FunctionCallsDropped)
}

func TestLimiter_Add_OneHundredThousandEvents(t *testing.T) {
	const retained = 128

	limiter := NewLimiter(retained, RuntimePHPFPM)
	tr := trace.Trace{}
	name := []byte("hot_path\x00")

	for i := 0; i < 100_000; i++ {
		limiter.Add(&tr, name, time.Duration(i), time.Nanosecond, int64(i))
	}

	assert.Len(t, tr.FunctionCalls, retained)
	assert.Equal(t, 100_000-retained, tr.FunctionCallsDropped)
	assert.Equal(t, int64(99_999), tr.ResourceUtilisation.MaxMemory)
	assert.LessOrEqual(t, cap(tr.FunctionCalls), retained*2)
}

func BenchmarkLimiter_OneHundredThousandEvents(b *testing.B) {
	name := []byte("hot_path\x00")

	for _, retained := range []int{100, 1_000} {
		b.Run(fmt.Sprintf("retained_%d", retained), func(b *testing.B) {
			b.ReportAllocs()

			for range b.N {
				limiter := NewLimiter(retained, RuntimePHPFPM)
				tr := trace.Trace{}

				for i := 0; i < 100_000; i++ {
					limiter.Add(&tr, name, time.Duration(i), time.Nanosecond, int64(i))
				}
			}
		})
	}
}

// A function called twice is one string, not two: a request repeats a few
// hundred names thousands of times, and allocating each repeat was an
// allocation per function call.
func TestLimiter_ReusesNames(t *testing.T) {
	limiter := NewLimiter(10, RuntimePHPFPM)
	tr := trace.Trace{}

	// Separate backing arrays, so a shared string can only come from the
	// limiter holding on to the first one.
	first := []byte("Drupal\\Core\\Entity::loadMultiple\x00")
	second := []byte("Drupal\\Core\\Entity::loadMultiple\x00")

	require.True(t, limiter.Add(&tr, first, 0, time.Nanosecond, 1))
	require.True(t, limiter.Add(&tr, second, 1, time.Nanosecond, 1))
	require.True(t, limiter.Add(&tr, []byte("Drupal\\Core\\Render::render\x00"), 2, time.Nanosecond, 1))

	require.Len(t, tr.FunctionCalls, 3)
	assert.Equal(t, "Drupal\\Core\\Entity::loadMultiple", tr.FunctionCalls[0].Name)
	assert.Equal(t, "Drupal\\Core\\Render::render", tr.FunctionCalls[2].Name)
	assert.Same(t,
		unsafe.StringData(tr.FunctionCalls[0].Name),
		unsafe.StringData(tr.FunctionCalls[1].Name),
		"the same function was named twice",
	)
}

// The names come from the application, so a workload which generates them at
// runtime must not grow the limiter without bound. Past the bound the names
// are still reported.
func TestLimiter_BoundsTheNamesItHolds(t *testing.T) {
	limiter := NewLimiter(MaxNames*2, RuntimePHPFPM)
	tr := trace.Trace{}

	for i := range MaxNames + 10 {
		require.True(t, limiter.Add(&tr, []byte(fmt.Sprintf("generated_%d\x00", i)), 0, time.Nanosecond, 1))
	}

	assert.Len(t, limiter.names, MaxNames)
	require.Len(t, tr.FunctionCalls, MaxNames+10)
	assert.Equal(t, fmt.Sprintf("generated_%d", MaxNames+9), tr.FunctionCalls[MaxNames+9].Name)
}

// A limiter which was not made by NewLimiter has nowhere to hold names, and
// has to keep naming calls rather than panicking on a nil map.
func TestLimiter_WithoutANameMap(t *testing.T) {
	limiter := Limiter{max: 10, runtime: RuntimePHPFPM}
	tr := trace.Trace{}

	require.True(t, limiter.Add(&tr, []byte("first\x00"), 0, time.Nanosecond, 1))
	require.True(t, limiter.Add(&tr, []byte("first\x00"), 1, time.Nanosecond, 1))

	require.Len(t, tr.FunctionCalls, 2)
	assert.Equal(t, "first", tr.FunctionCalls[1].Name)
}
