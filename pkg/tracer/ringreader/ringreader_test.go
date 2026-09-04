package ringreader

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cilium/ebpf/ringbuf"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skpr/compass/pkg/tracer/ingest"
)

type fakeReader struct {
	readFn  func() (ringbuf.Record, error)
	closeFn func() error
	reads   atomic.Int32
	closes  atomic.Int32
}

func (r *fakeReader) Read() (ringbuf.Record, error) {
	r.reads.Add(1)
	return r.readFn()
}

func (r *fakeReader) Close() error {
	r.closes.Add(1)
	if r.closeFn == nil {
		return nil
	}
	return r.closeFn()
}

func TestRun_CancellationClosesReaderWithoutReadError(t *testing.T) {
	started := make(chan struct{})
	closed := make(chan struct{})
	var startOnce sync.Once
	var closeOnce sync.Once
	reader := &fakeReader{
		readFn: func() (ringbuf.Record, error) {
			startOnce.Do(func() { close(started) })
			<-closed
			return ringbuf.Record{}, ringbuf.ErrClosed
		},
		closeFn: func() error {
			closeOnce.Do(func() { close(closed) })
			return nil
		},
	}
	counter := metricReadErrors.WithLabelValues(string(ingest.RuntimePHPCLI), string(StreamEvents))
	before := testutil.ToFloat64(counter)
	ctx, cancel := context.WithCancel(t.Context())
	result := make(chan error, 1)

	go func() {
		result <- Run(ctx, Source{
			Reader: reader, Runtime: ingest.RuntimePHPCLI, Stream: StreamEvents,
			Handle: func(context.Context, []byte) error { return nil },
		})
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("reader did not start")
	}
	cancel()

	select {
	case err := <-result:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("reader did not stop after cancellation")
	}
	assert.Equal(t, int32(1), reader.reads.Load())
	assert.GreaterOrEqual(t, reader.closes.Load(), int32(1))
	assert.Equal(t, before, testutil.ToFloat64(counter))
}

func TestRun_UnexpectedReadErrorIsFatalAndCountedOnce(t *testing.T) {
	readErr := errors.New("synthetic read failure")
	reader := &fakeReader{
		readFn: func() (ringbuf.Record, error) {
			return ringbuf.Record{}, readErr
		},
	}
	counter := metricReadErrors.WithLabelValues(string(ingest.RuntimeNodeHTTP), string(StreamEvents))
	before := testutil.ToFloat64(counter)

	err := Run(t.Context(), Source{
		Reader: reader, Runtime: ingest.RuntimeNodeHTTP, Stream: StreamEvents,
		Handle: func(context.Context, []byte) error { return nil },
	})

	require.ErrorIs(t, err, readErr)
	assert.ErrorContains(t, err, "runtime=node_http")
	assert.ErrorContains(t, err, "stream=events")
	assert.Equal(t, int32(1), reader.reads.Load(), "the reader must not retry a persistent error")
	assert.Equal(t, before+1, testutil.ToFloat64(counter))
}

func TestRun_UnexpectedCloseIsFatal(t *testing.T) {
	reader := &fakeReader{
		readFn: func() (ringbuf.Record, error) {
			return ringbuf.Record{}, ringbuf.ErrClosed
		},
	}
	counter := metricReadErrors.WithLabelValues(string(ingest.RuntimePHPFPM), string(StreamEvents))
	before := testutil.ToFloat64(counter)

	err := Run(t.Context(), Source{
		Reader: reader, Runtime: ingest.RuntimePHPFPM, Stream: StreamEvents,
		Handle: func(context.Context, []byte) error { return nil },
	})

	require.ErrorIs(t, err, ringbuf.ErrClosed)
	assert.Equal(t, before+1, testutil.ToFloat64(counter))
}

func TestRun_FPMReadErrorClosesSiblingReader(t *testing.T) {
	siblingStarted := make(chan struct{})
	siblingClosed := make(chan struct{})
	var startOnce sync.Once
	var closeOnce sync.Once
	sibling := &fakeReader{
		readFn: func() (ringbuf.Record, error) {
			startOnce.Do(func() { close(siblingStarted) })
			<-siblingClosed
			return ringbuf.Record{}, ringbuf.ErrClosed
		},
		closeFn: func() error {
			closeOnce.Do(func() { close(siblingClosed) })
			return nil
		},
	}
	readErr := errors.New("FPM events reader failed")
	failed := &fakeReader{
		readFn: func() (ringbuf.Record, error) {
			<-siblingStarted
			return ringbuf.Record{}, readErr
		},
	}

	err := Run(t.Context(),
		Source{
			Reader: failed, Runtime: ingest.RuntimePHPFPM, Stream: StreamEvents,
			Handle: func(context.Context, []byte) error { return nil },
		},
		Source{
			Reader: sibling, Runtime: ingest.RuntimePHPFPM, Stream: StreamDrupalCache,
			Handle: func(context.Context, []byte) error { return nil },
		},
	)

	require.ErrorIs(t, err, readErr)
	assert.GreaterOrEqual(t, sibling.closes.Load(), int32(1))
	select {
	case <-siblingClosed:
	default:
		t.Fatal("FPM sibling reader was not closed")
	}
}

func TestRun_ValidationFailureClosesReaders(t *testing.T) {
	valid := &fakeReader{
		readFn: func() (ringbuf.Record, error) {
			return ringbuf.Record{}, ringbuf.ErrClosed
		},
	}
	// A nil Handle fails validation, so Run rejects the sources before taking
	// ownership. The readers handed in must still be closed rather than leaked.
	invalid := &fakeReader{
		readFn: func() (ringbuf.Record, error) {
			return ringbuf.Record{}, ringbuf.ErrClosed
		},
	}

	err := Run(t.Context(),
		Source{
			Reader: valid, Runtime: ingest.RuntimePHPFPM, Stream: StreamEvents,
			Handle: func(context.Context, []byte) error { return nil },
		},
		Source{
			Reader: invalid, Runtime: ingest.RuntimePHPFPM, Stream: StreamDrupalCache,
			Handle: nil,
		},
	)

	require.Error(t, err)
	assert.Equal(t, int32(1), valid.closes.Load(), "the valid reader must be closed on validation failure")
	assert.Equal(t, int32(1), invalid.closes.Load(), "the invalid reader must be closed on validation failure")
}
