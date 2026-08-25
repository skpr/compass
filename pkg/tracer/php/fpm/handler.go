package fpm

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/patrickmn/go-cache"
	"golang.org/x/sys/unix"

	"github.com/skpr/compass/pkg/trace"
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
type Handler struct {
	// Consider an interface for the storage.
	storage *cache.Cache
	// Plugin for sending completed requests to.
	plugin sink.Interface
	// Options for the Handler eg. Thresholds.
	options Options
}

// Options for configuring the Handler.
type Options struct {
	Expire time.Duration
	// MaxCacheEvents is how many distinct Drupal cache events a trace retains.
	// Defaults to DefaultMaxCacheEvents.
	MaxCacheEvents int
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

	client := &Handler{
		storage: cache.New(options.Expire, options.Expire),
		plugin:  plugin,
		options: options,
	}

	return client, nil
}

// Handle the event and process it.
func (c *Handler) Handle(event bpfEvent) error {
	var (
		requestID = unix.ByteSliceToString(event.RequestId[:])
	)

	if requestID == "" {
		return fmt.Errorf("empty request id")
	}

	switch event.Type {
	case EventRequestInit:
		var (
			uri    = unix.ByteSliceToString(event.Uri[:])
			method = unix.ByteSliceToString(event.Method[:])
		)

		if err := c.handleRequestInit(requestID, uri, method, event); err != nil {
			return fmt.Errorf("failed to process request init: %w", err)
		}
	case EventFunction:
		if err := c.handleFunction(requestID, event); err != nil {
			return fmt.Errorf("failed to process function: %w", err)
		}
	case EventRequestShutdown:
		if err := c.handleRequestShutdown(requestID, event); err != nil {
			return fmt.Errorf("failed to process request shutdown: %w", err)
		}
	}

	return nil
}

// HandleDrupalCache event and process it.
//
// Drupal cache events arrive on their own ring buffer, with their own event
// type, so they enter the handler separately from the request lifecycle.
func (c *Handler) HandleDrupalCache(event bpfDrupalCacheEvent) error {
	requestID := unix.ByteSliceToString(event.RequestId[:])

	if requestID == "" {
		return fmt.Errorf("empty request id")
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

// Process the function event and store the data.
func (c *Handler) handleRequestInit(requestID, uri, method string, event bpfEvent) error {
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
				StartTime: int64(event.Timestamp),
			},
		},
		cacheIndex: make(map[string]int),
	}

	c.storage.Set(requestID, s, cache.DefaultExpiration)

	return nil
}

// Process the function event and store the data.
func (c *Handler) handleFunction(requestID string, event bpfEvent) error {
	function := trace.FunctionCall{
		Name: unix.ByteSliceToString(event.FunctionName[:]),
		// The start time is the event time minus how long it look to execute.
		// The event is triggerd after a the function is called and we have collected the elapsed time.
		StartTime: int64(event.Timestamp - event.Elapsed),
		Elapsed:   int64(event.Elapsed),
		Memory:    int64(event.Memory),
	}

	s, err := c.get(requestID)
	if err != nil {
		return err
	}

	if s.trace.ResourceUtilisation.MaxMemory < function.Memory {
		s.trace.ResourceUtilisation.MaxMemory = function.Memory
	}

	s.trace.FunctionCalls = append(s.trace.FunctionCalls, function)

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
		StartTime:  int64(event.Timestamp),
		Calls:      1,
	})

	c.touch(requestID, s)

	return nil
}

// Process the request shutdown event and send the profile to the plugin.
func (c *Handler) handleRequestShutdown(requestID string, event bpfEvent) error {
	s, err := c.get(requestID)
	if err != nil {
		return err
	}

	s.trace.Metadata.EndTime = int64(event.Timestamp)

	// Cleanup this request after we have processed it.
	defer c.storage.Delete(requestID)

	if len(s.trace.FunctionCalls) == 0 && s.trace.Drupal == nil {
		return fmt.Errorf("no functions found for request with id: %s", requestID)
	}

	err = c.plugin.ProcessTrace(context.TODO(), s.trace)
	if err != nil {
		return fmt.Errorf("failed to send profile data to plugin: %w", err)
	}

	return nil
}

// get the state of a request which is still being assembled.
func (c *Handler) get(requestID string) (*state, error) {
	x, found := c.storage.Get(requestID)
	if !found {
		return nil, fmt.Errorf("not found in storage")
	}

	s, ok := x.(*state)
	if !ok {
		return nil, fmt.Errorf("unexpected type in storage for request with id: %s", requestID)
	}

	return s, nil
}

// touch the stored request so that its expiry is measured from the last event
// it received rather than from when it started.
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
