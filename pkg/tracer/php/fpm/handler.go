package fpm

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
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
	// EventDrupalCacheRenderArray is the event type for cacheability derived from a render array.
	EventDrupalCacheRenderArray uint8 = 3
	// EventDrupalCacheObject is the event type for cacheability derived from an object.
	EventDrupalCacheObject uint8 = 4
)

// RequestID is the request id an event is tracked by: what the probe reported,
// normalised by requestKey.
//
// The storage is keyed on this rather than on a string made from it. Every
// function event has to find its request, and converting the field to a string
// to do that allocated once per function call, a million times over on the
// requests this has to keep up with. An array is comparable, so it is a map key
// as it stands.
type RequestID = [101]uint8

// requestKey is the id field of an event, as something two events of the same
// request can be matched by.
//
// The probes write a NUL-terminated string into a fixed-size field of a ring
// buffer record which is handed to them uninitialised, and writing the string
// does not clear what is behind it. So the same request id arrives with
// different trailing bytes in every event it produces, and the field as it
// stands matches nothing -- not even itself. What the field says is the same
// every time, which is what this returns: the string, in a buffer which is
// zero after it.
func requestKey(raw RequestID) RequestID {
	var key RequestID

	length := bytes.IndexByte(raw[:], 0)
	if length < 0 {
		length = len(raw)
	}

	copy(key[:], raw[:length])

	return key
}

// DefaultMaxCacheEvents is how many distinct Drupal cache events a trace keeps
// when no limit has been configured.
//
// Drupal derives cacheability without a threshold in front of it, so a single
// page can produce a great many of these. Identical events collapse into one
// entry, but a page which produces thousands of distinct ones would otherwise
// grow the trace without bound and put all of that on the wire.
const DefaultMaxCacheEvents = 250

// Handler for handling events.
//
// The tracer reads its two ring buffers in separate goroutines, so every method
// here can be called concurrently with any other.
type Handler struct {
	// mu guards the storage and the state behind the pointers it hands out.
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

// Options for configuring the Handler.
type Options struct {
	Expire time.Duration
	// Spans is how the request's function calls are aggregated: how many
	// spans a trace carries, and how finely calls are placed in time. The
	// zero value uses the package defaults.
	Spans spans.Options
	// MaxCacheEvents is how many distinct Drupal cache events a trace retains.
	// Defaults to DefaultMaxCacheEvents.
	MaxCacheEvents int
	// Clock relates the monotonic timestamps the probes emit to the wall clock.
	// The zero value reads the offset from the system.
	Clock clock.Monotonic
}

// state of a request which is still being assembled.
//
// The cache events are indexed as they arrive because they are aggregated by
// their contents, and a page can emit far more of them than a linear scan over
// what has already been collected would want to look at.
type state struct {
	trace trace.Trace
	// spans is the request's function calls, aggregated as they arrive.
	spans      *spans.Builder
	cacheIndex map[string]int
}

// NewHandler creates a new handler for processing events and sending profiles to the sink.
func NewHandler(plugin sink.Interface, options Options) (*Handler, error) {
	if options.MaxCacheEvents <= 0 {
		options.MaxCacheEvents = DefaultMaxCacheEvents
	}

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
		spans:   spans.New(options.Spans, spans.RuntimePHPFPM),
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

		if err := c.handleRequestInit(requestKey(event.RequestId), uri, method, event.Timestamp); err != nil {
			return fmt.Errorf("failed to process request init: %w", err)
		}
	case EventFunction:
		if err := c.handleFunction(requestKey(event.RequestId), event.FunctionName[:], event.Timestamp, event.Elapsed, event.Memory); err != nil {
			return fmt.Errorf("failed to process function: %w", err)
		}
	case EventRequestShutdown:
		if err := c.handleRequestShutdown(ctx, requestKey(event.RequestId), event.Timestamp); err != nil {
			return fmt.Errorf("failed to process request shutdown: %w", err)
		}
	}

	return nil
}

// identified reports whether an event carries a request id at all. The field
// is NUL-terminated, so an empty one starts with the terminator.
func identified(id RequestID) bool {
	return id[0] != 0
}

// HandleRequestInit processes one compact FPM request-init record.
func (c *Handler) HandleRequestInit(_ context.Context, event bpfRequestInitEvent) error {
	if !identified(event.RequestId) {
		return fmt.Errorf("%w: empty request id", ingest.ErrInvalidIdentifier)
	}
	if err := c.handleRequestInit(
		requestKey(event.RequestId),
		unix.ByteSliceToString(event.Uri[:]),
		unix.ByteSliceToString(event.Method[:]),
		event.Timestamp,
	); err != nil {
		return fmt.Errorf("failed to process request init: %w", err)
	}
	return nil
}

// HandleFunction processes one compact FPM function record.
func (c *Handler) HandleFunction(_ context.Context, event bpfFunctionEvent) error {
	if !identified(event.RequestId) {
		return fmt.Errorf("%w: empty request id", ingest.ErrInvalidIdentifier)
	}
	if err := c.handleFunction(requestKey(event.RequestId), event.FunctionName[:], event.Timestamp, event.Elapsed, event.Memory); err != nil {
		return fmt.Errorf("failed to process function: %w", err)
	}
	return nil
}

// HandleRequestShutdown processes one compact FPM request-shutdown record.
func (c *Handler) HandleRequestShutdown(ctx context.Context, event bpfRequestShutdownEvent) error {
	if !identified(event.RequestId) {
		return fmt.Errorf("%w: empty request id", ingest.ErrInvalidIdentifier)
	}
	if err := c.handleRequestShutdown(ctx, requestKey(event.RequestId), event.Timestamp); err != nil {
		return fmt.Errorf("failed to process request shutdown: %w", err)
	}
	return nil
}

// HandleDrupalCache event and process it.
//
// Drupal cache events arrive on their own ring buffer, with their own event
// type, so they enter the handler separately from the request lifecycle.
func (c *Handler) HandleDrupalCache(_ context.Context, event bpfDrupalCacheEvent) error {
	if !identified(event.RequestId) {
		return fmt.Errorf("%w: empty request id", ingest.ErrInvalidIdentifier)
	}

	var origin trace.CacheOrigin

	switch event.Type {
	case EventDrupalCacheRenderArray:
		origin = trace.CacheOriginRenderArray
	case EventDrupalCacheObject:
		origin = trace.CacheOriginObject
	default:
		return fmt.Errorf("unknown drupal cache event type: %d", event.Type)
	}

	if err := c.handleDrupalCache(requestKey(event.RequestId), origin, event); err != nil {
		return fmt.Errorf("failed to process drupal cache event: %w", err)
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
				Runtime: trace.RuntimePHP,
				HTTP: trace.MetadataHTTP{
					URI:    uri,
					Method: method,
				},
				StartTime: c.options.Clock.Time(timestamp),
			},
		},
		spans:      c.spans.Request(),
		cacheIndex: make(map[string]int),
	}

	c.storage.Set(requestID, s, timestamp)

	return nil
}

// Process the function event and store the data.
func (c *Handler) handleFunction(requestID RequestID, functionName []byte, timestamp, elapsed, memory uint64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	s, err := c.get(requestID, timestamp)
	if err != nil {
		return err
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

// Process a Drupal cache event and aggregate it into the trace.
func (c *Handler) handleDrupalCache(requestID RequestID, origin trace.CacheOrigin, event bpfDrupalCacheEvent) error {
	var (
		caller     = unix.ByteSliceToString(event.Caller[:])
		objectType = unix.ByteSliceToString(event.ObjectType[:])
		tags       = unix.ByteSliceToString(event.Tags[:])
		contexts   = unix.ByteSliceToString(event.Contexts[:])
	)

	c.mu.Lock()
	defer c.mu.Unlock()

	s, err := c.get(requestID, event.Timestamp)
	if err != nil {
		return err
	}

	if s.trace.Drupal == nil {
		s.trace.Drupal = &trace.Drupal{}
	}

	// Keyed on the raw strings rather than the split lists, so that building the
	// key costs nothing on the far more common path where the event is a repeat
	// of one already collected.
	key := strings.Join([]string{
		string(origin),
		caller,
		objectType,
		strconv.FormatInt(event.MaxAge, 10),
		tags,
		contexts,
	}, "\x00")

	if index, ok := s.cacheIndex[key]; ok {
		s.trace.Drupal.CacheEvents[index].Calls++

		return nil
	}

	if len(s.trace.Drupal.CacheEvents) >= c.options.MaxCacheEvents {
		s.trace.Drupal.CacheEventsDropped++

		return nil
	}

	s.cacheIndex[key] = len(s.trace.Drupal.CacheEvents)

	s.trace.Drupal.CacheEvents = append(s.trace.Drupal.CacheEvents, trace.CacheEvent{
		Origin:     origin,
		Caller:     caller,
		ObjectType: objectType,
		MaxAge:     event.MaxAge,
		Tags:       splitList(tags),
		Contexts:   splitList(contexts),
		Offset:     c.offset(s.trace.Metadata, event.Timestamp),
		Calls:      1,
	})

	return nil
}

// Process the request shutdown event and send the profile to the plugin.
func (c *Handler) handleRequestShutdown(ctx context.Context, requestID RequestID, timestamp uint64) error {
	t, err := c.complete(requestID, timestamp)
	if err != nil {
		return err
	}

	// Sent without the lock held: a sink which blocks would otherwise stall the
	// other ring buffer reader.
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

	s, err := c.get(requestID, timestamp)
	if err != nil {
		return trace.Trace{}, err
	}

	s.trace.Metadata.EndTime = c.options.Clock.Time(timestamp)
	s.spans.Finish(&s.trace)

	// Cleanup this request after we have processed it.
	defer c.storage.Delete(requestID)

	if len(s.trace.Spans) == 0 && s.trace.Drupal == nil {
		return trace.Trace{}, fmt.Errorf("%w: no functions found for request with id: %s", ingest.ErrTraceEmpty, unix.ByteSliceToString(requestID[:]))
	}

	return s.trace, nil
}

// get the state of a request which is still being assembled, and record that
// the request is still alive: its expiry is measured from the last event it
// produced rather than from when it started, so a request which is still
// running is not dropped out from under itself.
//
// This is a pointer into the storage, so the caller must hold c.mu for as long
// as it uses it.
func (c *Handler) get(requestID RequestID, timestamp uint64) (*state, error) {
	s, found := c.storage.Get(requestID, timestamp)
	if !found {
		return nil, fmt.Errorf("%w: request %q not found in storage", ingest.ErrRequestNotTracked, unix.ByteSliceToString(requestID[:]))
	}

	return s, nil
}

// splitList of space delimited values from a probe. Drupal cache tags and
// contexts cannot themselves contain a space, so the delimiter is unambiguous.
func splitList(value string) []string {
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return nil
	}

	return fields
}
