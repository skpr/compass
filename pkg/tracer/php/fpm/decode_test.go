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
