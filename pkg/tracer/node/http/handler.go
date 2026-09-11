package http

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/sys/unix"

	"github.com/skpr/compass/pkg/trace"
	"github.com/skpr/compass/pkg/tracer/clock"
	"github.com/skpr/compass/pkg/tracer/ingest"
	"github.com/skpr/compass/pkg/tracer/requests"
	"github.com/skpr/compass/pkg/tracer/sink"
	"github.com/skpr/compass/pkg/tracer/spans"
)

const (
	// EventFunction is the event type for a function.
	EventFunction uint8 = 0
	// EventRequestInit is the event type for a request init.
	EventRequestInit uint8 = 1
	// EventRequestShutdown is the event type for a request shutdown.
	EventRequestShutdown uint8 = 2
)

// RequestID is the request id as the probe reports it: a fixed-size,
// NUL-terminated field.
//
// The storage is keyed on this rather than on a string made from it. Every
// function event has to find its request, and converting the field to a
// string to do that allocated once per function call. An array is comparable,
// so it is a map key as it stands.
type RequestID = [101]uint8

// identified reports whether an event carries a request id at all. The field
// is NUL-terminated, so an empty one starts with the terminator.
func identified(id RequestID) bool {
	return id[0] != 0
}

// Handler for handling events.
type Handler struct {
	// mu guards the storage and the traces behind the pointers it hands out.
	mu sync.Mutex
	// storage holds the requests which are still being assembled.
	storage *requests.Store[RequestID, state]
	// Plugin for sending completed requests to.
	plugin sink.Interface
	// Options for the Handler eg. Thresholds.
	options Options
	// spans aggregates each request's function calls as their events arrive.
	spans *spans.Aggregator
}

// state of a request which is still being assembled.
type state struct {
	trace trace.Trace
	// spans is the request's function calls, aggregated as they arrive.
	spans *spans.Builder
}

// Options for configuring the Handler.
type Options struct {
	Expire time.Duration
	// Spans is how the request's function calls are aggregated: how many
	// spans a trace carries, and how finely calls are placed in time. The
	// zero value uses the package defaults.
	Spans spans.Options
	// Clock relates the monotonic timestamps the probes emit to the wall clock.
	// The zero value reads the offset from the system.
	Clock clock.Monotonic
}

// NewHandler creates a new handler for processing events and sending profiles to the sink.
func NewHandler(plugin sink.Interface, options Options) (*Handler, error) {
	if options.Clock.Boot.IsZero() {
		systemClock, err := clock.System()
		if err != nil {
			return nil, err
		}

		options.Clock = systemClock
	}

	client := &Handler{
		storage: requests.New[RequestID, state](options.Expire),
		plugin:  plugin,
		options: options,
		spans:   spans.New(options.Spans, spans.RuntimeNodeHTTP),
	}

	return client, nil
}

// Handle the event and process it.
func (c *Handler) Handle(ctx context.Context, event bpfEvent) error {
	if !identified(event.RequestId) {
		return fmt.Errorf("%w: empty request id", ingest.ErrInvalidIdentifier)
	}

	switch event.Type {
	case EventRequestInit:
		var (
			uri    = unix.ByteSliceToString(event.Uri[:])
			method = unix.ByteSliceToString(event.Method[:])
		)

		if err := c.handleRequestInit(event.RequestId, uri, method, event.Timestamp); err != nil {
			return fmt.Errorf("failed to process request init: %w", err)
		}
	case EventFunction:
		if err := c.handleFunction(event.RequestId, event.FunctionName[:], event.Timestamp, event.Elapsed, event.Memory); err != nil {
			return fmt.Errorf("failed to process function: %w", err)
		}
	case EventRequestShutdown:
		if err := c.handleRequestShutdown(ctx, event.RequestId, event.Timestamp); err != nil {
			return fmt.Errorf("failed to process request shutdown: %w", err)
		}
	}

	return nil
}

// HandleRequestInit processes one compact HTTP request-init record.
func (c *Handler) HandleRequestInit(_ context.Context, event bpfRequestInitEvent) error {
	if !identified(event.RequestId) {
		return fmt.Errorf("%w: empty request id", ingest.ErrInvalidIdentifier)
	}
	if err := c.handleRequestInit(
		event.RequestId,
		unix.ByteSliceToString(event.Uri[:]),
		unix.ByteSliceToString(event.Method[:]),
		event.Timestamp,
	); err != nil {
		return fmt.Errorf("failed to process request init: %w", err)
	}
	return nil
}

// HandleFunction processes one compact HTTP function record.
func (c *Handler) HandleFunction(_ context.Context, event bpfFunctionEvent) error {
	if !identified(event.RequestId) {
		return fmt.Errorf("%w: empty request id", ingest.ErrInvalidIdentifier)
	}
	if err := c.handleFunction(event.RequestId, event.FunctionName[:], event.Timestamp, event.Elapsed, event.Memory); err != nil {
		return fmt.Errorf("failed to process function: %w", err)
	}
	return nil
}

// HandleRequestShutdown processes one compact HTTP request-shutdown record.
func (c *Handler) HandleRequestShutdown(ctx context.Context, event bpfRequestShutdownEvent) error {
	if !identified(event.RequestId) {
		return fmt.Errorf("%w: empty request id", ingest.ErrInvalidIdentifier)
	}
	if err := c.handleRequestShutdown(ctx, event.RequestId, event.Timestamp); err != nil {
		return fmt.Errorf("failed to process request shutdown: %w", err)
	}
	return nil
}

// offset of a probe timestamp into the request it belongs to.
//
// The probes report against the monotonic clock, and the trace places
// everything inside it relative to the request start, so the conversion to an
// instant is only worth doing for the two ends of the request itself.
func (c *Handler) offset(metadata trace.Metadata, timestamp uint64) time.Duration {
	return c.options.Clock.Time(timestamp).Sub(metadata.StartTime)
}

// Process the function event and store the data.
func (c *Handler) handleRequestInit(requestID RequestID, uri, method string, timestamp uint64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	s := &state{
		trace: trace.Trace{
			Metadata: trace.Metadata{
				ID:      unix.ByteSliceToString(requestID[:]),
				Source:  trace.SourceHTTP,
				Runtime: trace.RuntimeNode,
				HTTP: trace.MetadataHTTP{
					URI:    uri,
					Method: method,
				},
				StartTime: c.options.Clock.Time(timestamp),
			},
		},
		spans: c.spans.Request(),
	}

	c.storage.Set(requestID, s, timestamp)

	return nil
}

// Process the function event and store the data.
func (c *Handler) handleFunction(requestID RequestID, functionName []byte, timestamp, elapsed, memory uint64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	s, found := c.storage.Get(requestID, timestamp)
	if !found {
		return fmt.Errorf("%w: request %q not found in storage", ingest.ErrRequestNotTracked, unix.ByteSliceToString(requestID[:]))
	}

	s.spans.Add(
		functionName,
		// The call started at the event time minus how long it took to execute:
		// the probe fires once the function has returned and its elapsed time
		// has been collected.
		c.offset(s.trace.Metadata, timestamp-elapsed),
		time.Duration(elapsed),
		int64(memory),
	)

	return nil
}

// Process the request shutdown event and send the profile to the plugin.
func (c *Handler) handleRequestShutdown(ctx context.Context, requestID RequestID, timestamp uint64) error {
	t, err := c.complete(requestID, timestamp)
	if err != nil {
		return err
	}

	// Sent without the lock held, so that a sink which blocks does not stall
	// the reader behind it.
	if err := c.plugin.ProcessTrace(ctx, t); err != nil {
		return fmt.Errorf("failed to send profile data to plugin: %w", err)
	}

	return nil
}

// complete a request, removing it from storage so that the caller is left
// holding the only reference to its trace.
func (c *Handler) complete(requestID RequestID, timestamp uint64) (trace.Trace, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	s, found := c.storage.Get(requestID, timestamp)
	if !found {
		return trace.Trace{}, fmt.Errorf("%w: request %q not found in storage", ingest.ErrRequestNotTracked, unix.ByteSliceToString(requestID[:]))
	}

	s.trace.Metadata.EndTime = c.options.Clock.Time(timestamp)
	s.spans.Finish(&s.trace)

	// Cleanup this request after we have processed it.
	defer c.storage.Delete(requestID)

	if len(s.trace.Spans) == 0 {
		return trace.Trace{}, fmt.Errorf("%w: no functions found for request with id: %s", ingest.ErrTraceEmpty, unix.ByteSliceToString(requestID[:]))
	}

	return s.trace, nil
}
