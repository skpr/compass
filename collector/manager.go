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
	case EventFunction:
		if err := c.handleFunction(requestID, event); err != nil {
			return fmt.Errorf("failed to process function: %w", err)
		}
	case EventRequestShutdown:
		var (
			uri    = unix.ByteSliceToString(event.Uri[:])
			method = unix.ByteSliceToString(event.Method[:])
		)

		if err := c.handleRequestShutdown(requestID, uri, method, int64(event.Timestamp)); err != nil {
			return fmt.Errorf("failed to process request shutdown: %w", err)
		}
	}

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

	var calls []trace.FunctionCall

	if x, found := c.storage.Get(requestID); found {
		calls = x.([]trace.FunctionCall)
	}

	calls = append(calls, function)

	c.storage.Set(requestID, calls, cache.DefaultExpiration)

	return nil
}

// Process the request shutdown event and send the profile to the plugin.
func (c *Manager) handleRequestShutdown(requestID, uri, method string, endTime int64) error {
	c.logger.Debug("request shutdown event has been called", "request_id", requestID)

	var calls []trace.FunctionCall

	if x, found := c.storage.Get(requestID); found {
		calls = x.([]trace.FunctionCall)
	}

	// Cleanup this request after we have processed it.
	defer c.storage.Delete(requestID)

	if len(calls) == 0 {
		return fmt.Errorf("no functions found for request with id: %s", requestID)
	}

	trace := trace.Trace{
		Metadata: trace.Metadata{
			RequestID: requestID,
			URI:       uri,
			Method:    method,
			EndTime:   endTime,
		},
	}

	for _, call := range calls {
		if trace.Metadata.StartTime == 0 {
			trace.Metadata.StartTime = call.StartTime
		}

		if call.StartTime < trace.Metadata.StartTime {
			trace.Metadata.StartTime = call.StartTime
		}

		trace.FunctionCalls = append(trace.FunctionCalls, call)
	}

	c.logger.Debug("request event has associated functions", "count", len(trace.FunctionCalls))

	err := c.plugin.ProcessTrace(trace)
	if err != nil {
		return fmt.Errorf("failed to send profile data to plugin: %w", err)
	}

	return nil
}
