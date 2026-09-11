package requests_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skpr/compass/pkg/tracer/requests"
)

type request struct {
	calls int
}

const (
	second = uint64(time.Second)
	minute = uint64(time.Minute)
)

func TestStore_HoldsAndReturnsTheSameState(t *testing.T) {
	store := requests.New[string, request](time.Minute)

	store.Set("a", &request{}, 0)

	first, ok := store.Get("a", second)
	require.True(t, ok)
	first.calls++

	second, ok := store.Get("a", 2*second)
	require.True(t, ok)

	assert.Same(t, first, second, "the store handed back a copy")
	assert.Equal(t, 1, second.calls)
}

func TestStore_ReportsAnUnknownRequest(t *testing.T) {
	store := requests.New[string, request](time.Minute)

	value, ok := store.Get("missing", 0)

	assert.False(t, ok)
	assert.Nil(t, value)
}

func TestStore_Delete(t *testing.T) {
	store := requests.New[string, request](time.Minute)

	store.Set("a", &request{}, 0)
	store.Delete("a")

	_, ok := store.Get("a", 0)
	assert.False(t, ok)
	assert.Equal(t, 0, store.Len())
}

// A request which stops producing events is abandoned: the process went away,
// or the shutdown probe was missed. It must not be held for the life of the
// collector.
func TestStore_DropsAbandonedRequests(t *testing.T) {
	store := requests.New[string, request](time.Minute)

	store.Set("abandoned", &request{}, 0)
	require.Equal(t, 1, store.Len())

	// A later request is what triggers the sweep, so the abandoned one goes
	// and the new one stays.
	store.Set("fresh", &request{}, 2*minute)

	assert.Equal(t, 1, store.Len())
	_, ok := store.Get("abandoned", 2*minute)
	assert.False(t, ok)
	_, ok = store.Get("fresh", 2*minute)
	assert.True(t, ok)
}

// Expiry runs from the last event a request produced, so a long-running CLI
// command survives for as long as it keeps calling functions.
func TestStore_KeepsRequestsWhichAreStillProducingEvents(t *testing.T) {
	store := requests.New[string, request](time.Minute)

	store.Set("long", &request{}, 0)

	// An event every ten seconds for five minutes, with other requests
	// starting alongside it to drive the sweep.
	for elapsed := 10 * second; elapsed <= 5*minute; elapsed += 10 * second {
		value, ok := store.Get("long", elapsed)
		require.True(t, ok, "dropped at %d", elapsed)
		value.calls++

		store.Set(fmt.Sprintf("other-%d", elapsed), &request{}, elapsed)
		store.Delete(fmt.Sprintf("other-%d", elapsed))
	}

	value, ok := store.Get("long", 5*minute)
	require.True(t, ok)
	assert.Equal(t, 30, value.calls)
}

// Sweeping walks the map, so it runs at most once per expiry period rather
// than on every request which starts.
func TestStore_SweepsNoMoreThanOncePerPeriod(t *testing.T) {
	store := requests.New[string, request](time.Minute)

	// Inside the first period every request is kept, however many start.
	for i := range 60 {
		store.Set(fmt.Sprintf("request-%d", i), &request{}, uint64(i)*second)
	}

	require.Equal(t, 60, store.Len())

	// The first request to arrive after the period sweeps, and drops only
	// those which have been silent for a whole one: request-0 to request-30.
	store.Set("later", &request{}, 90*second)

	assert.Equal(t, 30, store.Len())

	// The next period has to pass before it walks the map again.
	store.Set("later-still", &request{}, 100*second)

	assert.Equal(t, 31, store.Len())
}

func TestStore_DefaultExpire(t *testing.T) {
	store := requests.New[string, request](0)

	store.Set("a", &request{}, 0)
	store.Set("b", &request{}, uint64(requests.DefaultExpire.Nanoseconds())*2)

	assert.Equal(t, 1, store.Len())
}

// Timestamps come from the monotonic clock and do not go backwards, but a
// store must not sweep everything if one ever does.
func TestStore_TimestampGoingBackwards(t *testing.T) {
	store := requests.New[string, request](time.Minute)

	store.Set("a", &request{}, 10*minute)
	store.Set("b", &request{}, 0)

	assert.Equal(t, 2, store.Len())
}

func TestStore_IntegerKeys(t *testing.T) {
	store := requests.New[int64, request](time.Minute)

	store.Set(4242, &request{}, 0)

	value, ok := store.Get(4242, 0)
	require.True(t, ok)
	assert.NotNil(t, value)
}

// BenchmarkGet is the per-function-call cost of the store: one lookup which
// also keeps the request alive.
func BenchmarkGet(b *testing.B) {
	store := requests.New[string, request](time.Minute)
	store.Set("0123456789abcdef0123456789abcdef", &request{}, 0)

	b.ReportAllocs()

	var now uint64

	for b.Loop() {
		now += 1000

		value, ok := store.Get("0123456789abcdef0123456789abcdef", now)
		if !ok {
			b.Fatal("missing")
		}

		value.calls++
	}
}
