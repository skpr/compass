package fpm

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/patrickmn/go-cache"
	"golang.org/x/sys/unix"

	"github.com/skpr/compass/pkg/trace"
	"github.com/skpr/compass/pkg/tracer/clock"
	"github.com/skpr/compass/pkg/tracer/functioncalls"
	"github.com/skpr/compass/pkg/tracer/ingest"
	"github.com/skpr/compass/pkg/tracer/sink"
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
	// mu guards the state behind the pointers held in storage.
	mu sync.Mutex
	// Consider an interface for the storage.
	storage *cache.Cache
	// Plugin for sending completed requests to.
	plugin sink.Interface
	// Options for the Handler eg. Thresholds.
	options       Options
	functionCalls functioncalls.Limiter
}

// Options for configuring the Handler.
type Options struct {
	Expire time.Duration
	// MaxFunctionCalls is how many function records each trace retains.
	// Non-positive values use functioncalls.DefaultMax.
	MaxFunctionCalls int
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
	trace      trace.Trace
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
		storage:       cache.New(options.Expire, options.Expire),
		plugin:        plugin,
		options:       options,
		functionCalls: functioncalls.NewLimiter(options.MaxFunctionCalls, functioncalls.RuntimePHPFPM),
	}

	return client, nil
}

// Handle the event and process it.
func (c *Handler) Handle(ctx context.Context, event bpfEvent) error {
	var (
		requestID = unix.ByteSliceToString(event.RequestId[:])
	)

	if requestID == "" {
		return fmt.Errorf("%w: empty request id", ingest.ErrInvalidIdentifier)
	}

	switch event.Type {
	case EventRequestInit:
		var (
			uri    = unix.ByteSliceToString(event.Uri[:])
			method = unix.ByteSliceToString(event.Method[:])
		)

		if err := c.handleRequestInit(requestID, uri, method, event.Timestamp); err != nil {
			return fmt.Errorf("failed to process request init: %w", err)
		}
	case EventFunction:
		if err := c.handleFunction(requestID, event.FunctionName[:], event.Timestamp, event.Elapsed, event.Memory); err != nil {
			return fmt.Errorf("failed to process function: %w", err)
		}
	case EventRequestShutdown:
		if err := c.handleRequestShutdown(ctx, requestID, event.Timestamp); err != nil {
			return fmt.Errorf("failed to process request shutdown: %w", err)
		}
	}

	return nil
}

// HandleRequestInit processes one compact FPM request-init record.
func (c *Handler) HandleRequestInit(_ context.Context, event bpfRequestInitEvent) error {
	requestID := unix.ByteSliceToString(event.RequestId[:])
	if requestID == "" {
		return fmt.Errorf("%w: empty request id", ingest.ErrInvalidIdentifier)
	}
	if err := c.handleRequestInit(
		requestID,
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
	requestID := unix.ByteSliceToString(event.RequestId[:])
	if requestID == "" {
		return fmt.Errorf("%w: empty request id", ingest.ErrInvalidIdentifier)
	}
	if err := c.handleFunction(requestID, event.FunctionName[:], event.Timestamp, event.Elapsed, event.Memory); err != nil {
		return fmt.Errorf("failed to process function: %w", err)
	}
	return nil
}

// HandleRequestShutdown processes one compact FPM request-shutdown record.
func (c *Handler) HandleRequestShutdown(ctx context.Context, event bpfRequestShutdownEvent) error {
	requestID := unix.ByteSliceToString(event.RequestId[:])
	if requestID == "" {
		return fmt.Errorf("%w: empty request id", ingest.ErrInvalidIdentifier)
	}
	if err := c.handleRequestShutdown(ctx, requestID, event.Timestamp); err != nil {
		return fmt.Errorf("failed to process request shutdown: %w", err)
	}
	return nil
}

// HandleDrupalCache event and process it.
//
// Drupal cache events arrive on their own ring buffer, with their own event
// type, so they enter the handler separately from the request lifecycle.
func (c *Handler) HandleDrupalCache(_ context.Context, event bpfDrupalCacheEvent) error {
	requestID := unix.ByteSliceToString(event.RequestId[:])

	if requestID == "" {
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

	if err := c.handleDrupalCache(requestID, origin, event); err != nil {
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
func (c *Handler) handleRequestInit(requestID, uri, method string, timestamp uint64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	s := &state{
		trace: trace.Trace{
			Metadata: trace.Metadata{
				ID:      requestID,
				Source:  trace.SourceHTTP,
				Runtime: trace.RuntimePHP,
				HTTP: trace.MetadataHTTP{
					URI:    uri,
					Method: method,
				},
				StartTime: c.options.Clock.Time(timestamp),
			},
		},
		cacheIndex: make(map[string]int),
	}

	c.storage.Set(requestID, s, cache.DefaultExpiration)

	return nil
}

// Process the function event and store the data.
func (c *Handler) handleFunction(requestID string, functionName []byte, timestamp, elapsed, memory uint64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	s, err := c.get(requestID)
	if err != nil {
		return err
	}

	c.functionCalls.Add(
		&s.trace,
		functionName,
		// The call started at the event time minus how long it took to execute:
		// the probe fires once the function has returned and its elapsed time
		// has been collected.
		c.offset(s.trace.Metadata, timestamp-elapsed),
		time.Duration(elapsed),
		int64(memory),
	)

	c.touch(requestID, s)

	return nil
}

// Process a Drupal cache event and aggregate it into the trace.
func (c *Handler) handleDrupalCache(requestID string, origin trace.CacheOrigin, event bpfDrupalCacheEvent) error {
	var (
		caller     = unix.ByteSliceToString(event.Caller[:])
		objectType = unix.ByteSliceToString(event.ObjectType[:])
		tags       = unix.ByteSliceToString(event.Tags[:])
		contexts   = unix.ByteSliceToString(event.Contexts[:])
	)

	c.mu.Lock()
	defer c.mu.Unlock()

	s, err := c.get(requestID)
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
		c.touch(requestID, s)

		return nil
	}

	if len(s.trace.Drupal.CacheEvents) >= c.options.MaxCacheEvents {
		s.trace.Drupal.CacheEventsDropped++
		c.touch(requestID, s)

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

	c.touch(requestID, s)

	return nil
}

// Process the request shutdown event and send the profile to the plugin.
func (c *Handler) handleRequestShutdown(ctx context.Context, requestID string, timestamp uint64) error {
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
func (c *Handler) complete(requestID string, timestamp uint64) (trace.Trace, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	s, err := c.get(requestID)
	if err != nil {
		return trace.Trace{}, err
	}

	s.trace.Metadata.EndTime = c.options.Clock.Time(timestamp)

	// Cleanup this request after we have processed it.
	defer c.storage.Delete(requestID)

	if len(s.trace.FunctionCalls) == 0 && s.trace.Drupal == nil {
		return trace.Trace{}, fmt.Errorf("%w: no functions found for request with id: %s", ingest.ErrTraceEmpty, requestID)
	}

	return s.trace, nil
}

// get the state of a request which is still being assembled. This is a pointer
// into the storage, so the caller must hold c.mu for as long as it uses it.
func (c *Handler) get(requestID string) (*state, error) {
	x, found := c.storage.Get(requestID)
	if !found {
		return nil, fmt.Errorf("%w: request %q not found in storage", ingest.ErrRequestNotTracked, requestID)
	}

	s, ok := x.(*state)
	if !ok {
		return nil, fmt.Errorf("unexpected type in storage for request with id: %s", requestID)
	}

	return s, nil
}

// touch the stored request so that its expiry is measured from the last event
// it received rather than from when it started. The caller must hold c.mu.
func (c *Handler) touch(requestID string, s *state) {
	c.storage.Set(requestID, s, cache.DefaultExpiration)
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
