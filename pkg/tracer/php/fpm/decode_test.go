package fpm

import (
	"encoding/binary"
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skpr/compass/pkg/tracer/ingest"
)

// decodeFunctionEvent reads fields at offsets which were written by hand from
// the generated layout, so the guarantee it needs is that it produces exactly
// what a decode of that layout produces. Random samples exercise every byte,
// including the ones the offsets are not supposed to read.
func TestDecodeFunctionEvent_AgreesWithTheGeneratedLayout(t *testing.T) {
	require.Equal(t, functionEventSize, binary.Size(bpfFunctionEvent{}))

	random := rand.New(rand.NewPCG(1, 2))
	rawSample := make([]byte, functionEventSize)

	for range 2000 {
		for i := range rawSample {
			rawSample[i] = byte(random.UintN(256))
		}

		want, err := ingest.DecodeExact[bpfFunctionEvent](rawSample)
		require.NoError(t, err)

		got, err := decodeFunctionEvent(rawSample)
		require.NoError(t, err)

		require.Equal(t, want, got)
	}
}

// A sample of the wrong length means the layout is not the one the offsets were
// written for, which has to be an error rather than a misread field.
func TestDecodeFunctionEvent_RejectsTheWrongSize(t *testing.T) {
	for _, size := range []int{0, 1, functionEventSize - 1, functionEventSize + 1} {
		_, err := decodeFunctionEvent(make([]byte, size))
		assert.ErrorContains(t, err, "invalid event size")
	}
}

// BenchmarkDecodeFunctionEventPath compares the two decodes of the record which
// arrives once per function call.
func BenchmarkDecodeFunctionEventPath(b *testing.B) {
	rawSample := make([]byte, functionEventSize)
	rawSample[0] = EventFunction
	copy(rawSample[1:102], "0123456789abcdef0123456789abcdef")
	copy(rawSample[102:203], "Drupal\\Core\\Entity\\Sql\\SqlContentEntityStorage::loadMultiple")
	binary.LittleEndian.PutUint64(rawSample[208:216], 1700000000)
	binary.LittleEndian.PutUint64(rawSample[216:224], 5000)
	binary.LittleEndian.PutUint64(rawSample[224:232], 1<<20)

	b.Run("reflection", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(rawSample)))

		for b.Loop() {
			if _, err := ingest.DecodeExact[bpfFunctionEvent](rawSample); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("offsets", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(rawSample)))

		for b.Loop() {
			if _, err := decodeFunctionEvent(rawSample); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// dirtyRecord is a ring-buffer record as the kernel hands one over: the fields
// a probe wrote, and whatever the previous record left everywhere else.
func dirtyRecord(size int, eventType uint8, noise uint8, fields map[int]string) []byte {
	raw := make([]byte, size)

	for i := range raw {
		raw[i] = noise
	}

	raw[0] = eventType

	for offset, value := range fields {
		copy(raw[offset:], value)
		raw[offset+len(value)] = 0
	}

	return raw
}

// The whole path a request takes, from records which carry rubbish behind
// every string, as the ring buffer really delivers them. A request has to come
// out of it whole: the events are only related to each other by an id field
// whose trailing bytes differ in each one.
func TestProcessEvent_AssemblesARequestFromDirtyRecords(t *testing.T) {
	sink := &mockSink{}
	handler, err := NewHandler(sink, testOptions())
	require.NoError(t, err)

	skips := ingest.NewSkips(ingest.RuntimePHPFPM)

	init := dirtyRecord(2216, EventRequestInit, 0x7F, map[int]string{
		1:   "0123456789abcdef0123456789abcdef",
		102: "GET",
		203: "/node/1",
	})
	binary.LittleEndian.PutUint64(init[2208:2216], 1000)

	function := dirtyRecord(functionEventSize, EventFunction, 0xA5, map[int]string{
		1:   "0123456789abcdef0123456789abcdef",
		102: "Drupal\\Core\\Entity::loadMultiple",
	})
	binary.LittleEndian.PutUint64(function[208:216], 1500)
	binary.LittleEndian.PutUint64(function[216:224], 200)
	binary.LittleEndian.PutUint64(function[224:232], 4096)

	shutdown := dirtyRecord(112, EventRequestShutdown, 0x3C, map[int]string{
		1: "0123456789abcdef0123456789abcdef",
	})
	binary.LittleEndian.PutUint64(shutdown[104:112], 2000)

	for _, raw := range [][]byte{init, function, shutdown} {
		require.NoError(t, processEvent(t.Context(), raw, handler, skips))
	}

	assert.Zero(t, skips.Total(), "an event was skipped")

	traces := sink.Traces()
	require.Len(t, traces, 1)
	assert.Equal(t, "0123456789abcdef0123456789abcdef", traces[0].Metadata.ID)
	assert.Equal(t, "GET", traces[0].Metadata.HTTP.Method)
	assert.Equal(t, "/node/1", traces[0].Metadata.HTTP.URI)
	require.Len(t, traces[0].Spans, 1)
	assert.Equal(t, "Drupal\\Core\\Entity::loadMultiple", traces[0].Spans[0].Name)
	assert.Equal(t, int64(1), traces[0].Calls)
	assert.Equal(t, int64(4096), traces[0].ResourceUtilisation.MaxMemory)
}
