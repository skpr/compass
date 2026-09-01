package ingest

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testEvent struct {
	Outcome uint8
}

func TestSkips_Record(t *testing.T) {
	skips := NewSkips(RuntimeNodeHTTP)
	counter := metricEventsSkipped.WithLabelValues(string(RuntimeNodeHTTP), string(ReasonRequestNotTracked))
	before := testutil.ToFloat64(counter)

	assert.True(t, skips.Record(errors.Join(errors.New("handler context"), ErrRequestNotTracked)))
	assert.Equal(t, uint64(1), skips.Count(ReasonRequestNotTracked))
	assert.Equal(t, uint64(1), skips.Total())
	assert.Equal(t, before+1, testutil.ToFloat64(counter))

	assert.False(t, skips.Record(errors.New("sink failed")))
	assert.Equal(t, uint64(1), skips.Total())
}

func TestNewSkips_RejectsUnknownRuntime(t *testing.T) {
	assert.Panics(t, func() {
		NewSkips(Runtime("user-supplied-label"))
	})
}

func TestDecodeAndHandle(t *testing.T) {
	raw := encodeTestEvent(t, testEvent{Outcome: 1})
	skips := NewSkips(RuntimePHPCLI)

	err := DecodeAndHandle(t.Context(), raw, func(_ context.Context, event testEvent) error {
		require.Equal(t, uint8(1), event.Outcome)
		return fmtWrap(ErrTraceEmpty)
	}, skips)

	require.NoError(t, err)
	assert.Equal(t, uint64(1), skips.Count(ReasonTraceEmpty))
}

func TestDecodeAndHandle_FatalErrors(t *testing.T) {
	t.Run("decode", func(t *testing.T) {
		err := DecodeAndHandle(t.Context(), []byte{}, func(_ context.Context, _ testEvent) error {
			t.Fatal("handler must not be called")
			return nil
		}, NewSkips(RuntimePHPFPM))
		assert.ErrorContains(t, err, "failed to read event")
	})

	t.Run("handler", func(t *testing.T) {
		sinkErr := errors.New("sink failed")
		err := DecodeAndHandle(t.Context(), encodeTestEvent(t, testEvent{}), func(_ context.Context, _ testEvent) error {
			return sinkErr
		}, NewSkips(RuntimePHPFPM))
		assert.ErrorIs(t, err, sinkErr)
		assert.ErrorContains(t, err, "failed to handle event")
	})
}

func encodeTestEvent(t *testing.T, event testEvent) []byte {
	t.Helper()

	var buf bytes.Buffer
	require.NoError(t, binary.Write(&buf, binary.LittleEndian, event))

	return buf.Bytes()
}

func fmtWrap(err error) error {
	return errors.Join(errors.New("wrapped"), err)
}
