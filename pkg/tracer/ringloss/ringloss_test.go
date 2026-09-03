package ringloss

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skpr/compass/pkg/tracer/ingest"
	"github.com/skpr/compass/pkg/tracer/ringreader"
)

type fakeCounterMap struct {
	values map[uint32][]uint64
}

func (m *fakeCounterMap) Lookup(key, valueOut interface{}) error {
	keyValue, ok := key.(uint32)
	if !ok {
		return fmt.Errorf("unexpected key type %T", key)
	}
	output, ok := valueOut.(*[]uint64)
	if !ok {
		return fmt.Errorf("unexpected output type %T", valueOut)
	}
	values, ok := m.values[keyValue]
	if !ok {
		return fmt.Errorf("key %d not found", keyValue)
	}
	*output = append((*output)[:0], values...)
	return nil
}

func TestObserver_ExportsExactPerCPUDeltas(t *testing.T) {
	counterMap := &fakeCounterMap{values: map[uint32][]uint64{
		keyEvents:      {2, 3, 5},
		keyDrupalCache: {7, 11, 13},
	}}
	counters := newTestCounters()
	var warnings []string
	observer, err := newObserver(counterMap, ingest.RuntimePHPFPM, counters,
		func(runtime ingest.Runtime, stream ringreader.Stream, failures uint64) {
			warnings = append(warnings, fmt.Sprintf("%s/%s=%d", runtime, stream, failures))
		},
		ringreader.StreamEvents,
		ringreader.StreamDrupalCache,
	)
	require.NoError(t, err)

	require.NoError(t, observer.Observe())
	assert.Equal(t, float64(10), counterValue(counters, ingest.RuntimePHPFPM, ringreader.StreamEvents))
	assert.Equal(t, float64(31), counterValue(counters, ingest.RuntimePHPFPM, ringreader.StreamDrupalCache))

	counterMap.values[keyEvents] = []uint64{3, 5, 8}
	counterMap.values[keyDrupalCache] = []uint64{7, 11, 14}
	require.NoError(t, observer.Observe())
	assert.Equal(t, float64(16), counterValue(counters, ingest.RuntimePHPFPM, ringreader.StreamEvents))
	assert.Equal(t, float64(32), counterValue(counters, ingest.RuntimePHPFPM, ringreader.StreamDrupalCache))
	assert.Equal(t, uint64(16), observer.Total(ringreader.StreamEvents))
	assert.Equal(t, uint64(32), observer.Total(ringreader.StreamDrupalCache))
	assert.ElementsMatch(t, []string{"php_fpm/events=10", "php_fpm/drupal_cache=31"}, warnings,
		"each stream warns once per run rather than once per event or poll")
}

func TestObserver_NewBPFObjectPreservesMetricWithoutDoubleCounting(t *testing.T) {
	counters := newTestCounters()
	newRun := func(values []uint64) *Observer {
		observer, err := newObserver(
			&fakeCounterMap{values: map[uint32][]uint64{keyEvents: values}},
			ingest.RuntimeNodeHTTP,
			counters,
			func(ingest.Runtime, ringreader.Stream, uint64) {},
			ringreader.StreamEvents,
		)
		require.NoError(t, err)
		return observer
	}

	firstRun := newRun([]uint64{4, 6})
	require.NoError(t, firstRun.Observe())
	require.NoError(t, firstRun.Observe())
	assert.Equal(t, float64(10), counterValue(counters, ingest.RuntimeNodeHTTP, ringreader.StreamEvents),
		"re-reading one BPF object must not add its cumulative value twice")

	secondRun := newRun([]uint64{1, 2})
	require.NoError(t, secondRun.Observe())
	require.NoError(t, secondRun.Observe())
	assert.Equal(t, float64(13), counterValue(counters, ingest.RuntimeNodeHTTP, ringreader.StreamEvents),
		"a recreated BPF object's fresh total is added to the process-level counter")
}

func TestObserver_RejectsCounterDecreaseWithinRun(t *testing.T) {
	counterMap := &fakeCounterMap{values: map[uint32][]uint64{keyEvents: {5}}}
	observer, err := newObserver(counterMap, ingest.RuntimePHPCLI, newTestCounters(),
		func(ingest.Runtime, ringreader.Stream, uint64) {}, ringreader.StreamEvents)
	require.NoError(t, err)
	require.NoError(t, observer.Observe())

	counterMap.values[keyEvents] = []uint64{4}
	err = observer.Observe()
	assert.ErrorContains(t, err, "counter decreased")
}

func newTestCounters() *prometheus.CounterVec {
	return prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "test_ringbuf_reserve_failures_total",
		Help: "Test counter.",
	}, []string{"runtime", "stream"})
}

func counterValue(counters *prometheus.CounterVec, runtime ingest.Runtime, stream ringreader.Stream) float64 {
	return testutil.ToFloat64(counters.WithLabelValues(string(runtime), string(stream)))
}

func TestBPFPrograms_CountEveryReserveFailure(t *testing.T) {
	tests := []struct {
		path        string
		events      int
		drupalCache int
	}{
		{path: "../node/http/program.bpf.c", events: 3},
		{path: "../php/cli/program.bpf.c", events: 3},
		{path: "../php/fpm/program.bpf.c", events: 3, drupalCache: 2},
	}

	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			source, err := os.ReadFile(test.path)
			require.NoError(t, err)
			program := string(source)
			reserves := strings.Count(program, "bpf_ringbuf_reserve(")
			eventCounts := strings.Count(program, "count_ringbuf_reserve_failure(RINGBUF_STREAM_EVENTS);")
			drupalCounts := strings.Count(program, "count_ringbuf_reserve_failure(RINGBUF_STREAM_DRUPAL_CACHE);")

			assert.Equal(t, test.events, eventCounts)
			assert.Equal(t, test.drupalCache, drupalCounts)
			assert.Equal(t, reserves, eventCounts+drupalCounts,
				"every reserve path must increment exactly one stream counter")
			assert.Contains(t, program, "BPF_MAP_TYPE_PERCPU_ARRAY")
		})
	}
}
