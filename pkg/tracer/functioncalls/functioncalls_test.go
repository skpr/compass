package functioncalls

import (
	"fmt"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"

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
