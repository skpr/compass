package collector

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/patrickmn/go-cache"
	"golang.org/x/sys/unix"

	"github.com/skpr/compass/collector/sink"
	"github.com/skpr/compass/profile/complete"
)

const (
	// EventFunction is the event type for a function.
	EventFunction = "function"
	// EventRequestShutdown is the event type for a request shutdown.
	EventRequestShutdown = "request_shutdown"
	// FunctionNameRoot is used to identify the root function for a requeust (it's an empty name).
	FunctionNameRoot = ""
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
		if err := c.handleRequestShutdown(requestID); err != nil {
			return fmt.Errorf("failed to process request shutdown: %w", err)
		}
	}

	return nil
}

// Process the function event and store the data.
func (c *Manager) handleFunction(requestID string, event bpfEvent) error {
	function := complete.FunctionCall{
		Name:      fmt.Sprintf("%s::%s", unix.ByteSliceToString(event.ClassName[:]), unix.ByteSliceToString(event.FunctionName[:])),
		StartTime: int64(event.StartTime),
		EndTime:   int64(event.EndTime),
	}

	c.logger.Debug("function event has been called",
		"request_id", requestID,
		"function_name", function.Name,
		"start_time", function.StartTime,
		"end_time", function.EndTime,
	)

	var calls []complete.FunctionCall

	if x, found := c.storage.Get(requestID); found {
		calls = x.([]complete.FunctionCall)
	}

	calls = append(calls, function)

	c.storage.Set(requestID, calls, cache.DefaultExpiration)

	return nil
}

// Process the request shutdown event and send the profile to the plugin.
func (c *Manager) handleRequestShutdown(requestID string) error {
	c.logger.Debug("request shutdown event has been called", "request_id", requestID)

	var calls []complete.FunctionCall

	if x, found := c.storage.Get(requestID); found {
		calls = x.([]complete.FunctionCall)
	}

	// Cleanup this request after we have processed it.
	defer c.storage.Delete(requestID)

	if len(calls) == 0 {
		return fmt.Errorf("no functions found for request with id: %s", requestID)
	}

	profile := complete.Profile{
		RequestID: requestID,
	}

	for _, call := range calls {
		if profile.StartTime == 0 {
			profile.StartTime = call.StartTime
		}

		if call.StartTime < profile.StartTime {
			profile.StartTime = call.StartTime
		}

		if call.EndTime > profile.EndTime {
			profile.EndTime = call.EndTime
		}

		profile.FunctionCalls = append(profile.FunctionCalls, call)
	}

	profile.ExecutionTime = (profile.EndTime - profile.StartTime) / 1000

	c.logger.Debug("request event has associated functions", "count", len(profile.FunctionCalls))

	err := c.plugin.ProcessProfile(profile)
	if err != nil {
		return fmt.Errorf("failed to send profile data to plugin: %w", err)
	}

	return nil
}
