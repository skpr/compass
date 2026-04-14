package http

import (
	"context"
	"fmt"
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
)

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
}

// NewHandler creates a new handler for processing events and sending profiles to the sink.
func NewHandler(plugin sink.Interface, options Options) (*Handler, error) {
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

// Process the function event and store the data.
func (c *Handler) handleRequestInit(requestID, uri, method string, event bpfEvent) error {
	t := trace.Trace{
		Metadata: trace.Metadata{
			ID:      requestID,
			Source:  trace.SourceHTTP,
			Runtime: trace.RuntimeNode,
			HTTP: trace.MetadataHTTP{
				URI:    uri,
				Method: method,
			},
			StartTime: int64(event.Timestamp),
		},
	}

	c.storage.Set(requestID, t, cache.DefaultExpiration)

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

	x, found := c.storage.Get(requestID)
	if !found {
		return fmt.Errorf("not found in storage")
	}

	t := x.(trace.Trace)

	if t.ResourceUtilisation.MaxMemory < function.Memory {
		t.ResourceUtilisation.MaxMemory = function.Memory
	}

	t.FunctionCalls = append(t.FunctionCalls, function)

	c.storage.Set(requestID, t, cache.DefaultExpiration)

	return nil
}

// Process the request shutdown event and send the profile to the plugin.
func (c *Handler) handleRequestShutdown(requestID string, event bpfEvent) error {
	x, found := c.storage.Get(requestID)
	if !found {
		return fmt.Errorf("not found in storage")
	}

	t := x.(trace.Trace)

	t.Metadata.EndTime = int64(event.Timestamp)

	// Cleanup this request after we have processed it.
	defer c.storage.Delete(requestID)

	if len(t.FunctionCalls) == 0 {
		return fmt.Errorf("no functions found for request with id: %s", requestID)
	}

	err := c.plugin.ProcessTrace(context.TODO(), t)
	if err != nil {
		return fmt.Errorf("failed to send profile data to plugin: %w", err)
	}

	return nil
}
