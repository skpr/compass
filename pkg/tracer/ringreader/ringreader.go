// Package ringreader coordinates ring-buffer readers for runtime tracers.
package ringreader

import (
	"context"
	"errors"
	"fmt"

	"github.com/cilium/ebpf/ringbuf"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"golang.org/x/sync/errgroup"

	"github.com/skpr/compass/pkg/tracer/ingest"
)

// Stream is a fixed metric label for a runtime ring buffer.
type Stream string

// Stream metric labels.
const (
	StreamEvents      Stream = "events"
	StreamDrupalCache Stream = "drupal_cache"
)

var metricReadErrors = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "compass_sidecar_ringbuf_read_errors_total",
	Help: "The total number of unexpected ring-buffer read errors.",
}, []string{"runtime", "stream"})

// Reader is the subset of ringbuf.Reader used by runtime tracers.
type Reader interface {
	Read() (ringbuf.Record, error)
	Close() error
}

// Source describes one runtime ring-buffer stream.
type Source struct {
	Reader  Reader
	Runtime ingest.Runtime
	Stream  Stream
	Handle  func(context.Context, []byte) error
}

// Run reads all sources until the context is cancelled or one source fails.
// A source failure cancels the group and closes every reader so blocked sibling
// reads return promptly.
func Run(ctx context.Context, sources ...Source) error {
	if len(sources) == 0 {
		return errors.New("at least one ring-buffer source is required")
	}

	for _, source := range sources {
		if err := validate(source); err != nil {
			return err
		}
	}

	group, groupCtx := errgroup.WithContext(ctx)

	for _, source := range sources {
		group.Go(func() error {
			return read(groupCtx, source)
		})
	}

	group.Go(func() error {
		<-groupCtx.Done()
		for _, source := range sources {
			_ = source.Reader.Close()
		}
		return nil
	})

	return group.Wait()
}

func read(ctx context.Context, source Source) error {
	for {
		if ctx.Err() != nil {
			return nil
		}

		record, err := source.Reader.Read()
		if err != nil {
			if errors.Is(err, ringbuf.ErrClosed) && ctx.Err() != nil {
				return nil
			}

			metricReadErrors.WithLabelValues(string(source.Runtime), string(source.Stream)).Inc()
			return fmt.Errorf("ring-buffer read failed (runtime=%s, stream=%s): %w", source.Runtime, source.Stream, err)
		}

		if err := source.Handle(ctx, record.RawSample); err != nil {
			return err
		}
	}
}

func validate(source Source) error {
	if source.Reader == nil {
		return fmt.Errorf("ring-buffer reader is required")
	}
	if source.Handle == nil {
		return fmt.Errorf("ring-buffer handler is required")
	}

	switch source.Runtime {
	case ingest.RuntimeNodeHTTP, ingest.RuntimePHPCLI:
		if source.Stream != StreamEvents {
			return fmt.Errorf("unknown ring-buffer stream %q for runtime %q", source.Stream, source.Runtime)
		}
	case ingest.RuntimePHPFPM:
		if source.Stream != StreamEvents && source.Stream != StreamDrupalCache {
			return fmt.Errorf("unknown ring-buffer stream %q for runtime %q", source.Stream, source.Runtime)
		}
	default:
		return fmt.Errorf("unknown ring-buffer runtime %q", source.Runtime)
	}

	return nil
}
