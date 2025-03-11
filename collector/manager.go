package collector

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/patrickmn/go-cache"
	"golang.org/x/sys/unix"

	"github.com/skpr/compass/collector/sink"
	"github.com/skpr/compass/trace"
)

const (
	// EventRequestInit is the event type for a request init.
	EventRequestInit = "request_init"
	// EventFunction is the event type for a function.
	EventFunction = "function"
	// EventRequestShutdown is the event type for a request shutdown.
	EventRequestShutdown = "request_shutdown"
)

// Manager for handling events.
type Manager struct {
	// Logger for debugging.
	logger *slog.Logger
	// Consider an interface for the storage.
	storage *cache.Cache
	// Plugin for sending completed requests to.
	plugin sink.Interface
	// Options for the manager eg. Thresholds.
	options Options
}

// Options for configuring the manager.
type Options struct {
	Expire time.Duration
}

// NewManager creates a new manager.
func NewManager(logger *slog.Logger, plugin sink.Interface, options Options) (*Manager, error) {
	client := &Manager{
		logger:  logger,
		storage: cache.New(options.Expire, options.Expire),
		plugin:  plugin,
		options: options,
	}

	return client, nil
}

// Handle the event and process it.
func (c *Manager) Handle(event bpfEvent) error {
	var (
		eventType = unix.ByteSliceToString(event.Type[:])
		requestID = unix.ByteSliceToString(event.RequestId[:])
	)

	if requestID == "" {
		return fmt.Errorf("empty request id")
	}

	switch eventType {
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
func (c *Manager) handleRequestInit(requestID, uri, method string, event bpfEvent) error {
	c.logger.Debug("request initialise event has been called", "request_id", requestID)

	t := trace.Trace{
		Metadata: trace.Metadata{
			RequestID: requestID,
			URI:       uri,
			Method:    method,
			StartTime: int64(event.Timestamp),
		},
	}

	c.storage.Set(requestID, t, cache.DefaultExpiration)

	return nil
}

// Process the function event and store the data.
func (c *Manager) handleFunction(requestID string, event bpfEvent) error {
	function := trace.FunctionCall{
		Name: unix.ByteSliceToString(event.FunctionName[:]),
		// The start time is the event time minus how long it look to execute.
		// The event is triggerd after a the function is called and we have collected the elapsed time.
		StartTime: int64(event.Timestamp - event.Elapsed),
		Elapsed:   int64(event.Elapsed),
	}

	c.logger.Debug("function event has been called",
		"request_id", requestID,
		"function_name", function.Name,
		"start_time", function.StartTime,
		"elapsed", function.Elapsed,
	)

	x, found := c.storage.Get(requestID)
	if !found {
		return fmt.Errorf("not found in storage")
	}

	t := x.(trace.Trace)

	t.FunctionCalls = append(t.FunctionCalls, function)

	c.storage.Set(requestID, t, cache.DefaultExpiration)

	return nil
}

// Process the request shutdown event and send the profile to the plugin.
func (c *Manager) handleRequestShutdown(requestID string, event bpfEvent) error {
	c.logger.Debug("request shutdown event has been called", "request_id", requestID)

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

	c.logger.Debug("request event has associated functions", "count", len(t.FunctionCalls))

	err := c.plugin.ProcessTrace(t)
	if err != nil {
		return fmt.Errorf("failed to send profile data to plugin: %w", err)
	}

	return nil
}
