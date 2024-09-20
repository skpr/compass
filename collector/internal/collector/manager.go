package collector

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/patrickmn/go-cache"
	"golang.org/x/sys/unix"

	"github.com/skpr/compass/collector/pkg/tracing"
	"github.com/skpr/compass/collector/plugin"
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
	plugin plugin.Interface
	// Options for the manager eg. Thresholds.
	options ManagerOptions
}

// ManagerOptions for configuring the manager.
type ManagerOptions struct {
	Expire            time.Duration
	RequestThreshold  float64
	FunctionThreshold float64
}

// NewManager creates a new manager.
func NewManager(logger *slog.Logger, plugin plugin.Interface, options ManagerOptions) (*Manager, error) {
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

	// We typically see this type of request ID for the PHP-FPM stats endpoint.
	if requestID == "UNKNOWN" {
		return fmt.Errorf("unknown request id: %s", requestID)
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
	function := tracing.Function{
		Name:          unix.ByteSliceToString(event.FunctionName[:]),
		ExecutionTime: float64(event.ExecutionTime) / 1e6,
	}

	c.logger.Debug("function event has been called",
		"request_id", requestID,
		"function_name", function.Name,
		"execution_time", function.ExecutionTime,
	)

	var functions []tracing.Function

	if x, found := c.storage.Get(requestID); found {
		functions = x.([]tracing.Function)
	}

	functions = append(functions, function)

	c.storage.Set(requestID, functions, cache.DefaultExpiration)

	return nil
}

// Process the request shutdown event and send the profile to the plugin.
func (c *Manager) handleRequestShutdown(requestID string) error {
	c.logger.Debug("request shutdown event has been called", "request_id", requestID)

	var functions []tracing.Function

	if x, found := c.storage.Get(requestID); found {
		functions = x.([]tracing.Function)
	}

	// Cleanup this request after we have processed it.
	defer c.storage.Delete(requestID)

	if len(functions) == 0 {
		return fmt.Errorf("no functions found for request with id: %s", requestID)
	}

	profile := tracing.Profile{
		RequestID: requestID,
		Functions: make(map[string]tracing.FunctionSummary),
	}

	for _, function := range functions {
		if function.Name == FunctionNameRoot {
			profile.ExecutionTime = function.ExecutionTime
			continue
		}

		f := tracing.FunctionSummary{
			TotalExecutionTime: function.ExecutionTime,
			Invocations:        1,
		}

		if _, ok := profile.Functions[function.Name]; ok {
			f.TotalExecutionTime = f.TotalExecutionTime + profile.Functions[function.Name].TotalExecutionTime
			f.Invocations = f.Invocations + profile.Functions[function.Name].Invocations
		}

		profile.Functions[function.Name] = f
	}

	c.logger.Debug("request event has associated functions", "count", len(profile.Functions))

	// Don't send if less than threshold.
	if profile.ExecutionTime < c.options.RequestThreshold {
		return nil
	}

	// Reduce the functions based on threshold.
	profile.Functions = reduceFunctions(profile.Functions, c.options.FunctionThreshold)

	err := c.plugin.ProcessProfile(profile)
	if err != nil {
		return fmt.Errorf("failed to send profile data to plugin: %w", err)
	}

	return nil
}

// Helper function to reduce the profile output for stdout and cut out unnecessary noise.
func reduceFunctions(functions map[string]tracing.FunctionSummary, threshold float64) map[string]tracing.FunctionSummary {
	var reduced = make(map[string]tracing.FunctionSummary)

	for name, function := range functions {
		if function.TotalExecutionTime > threshold {
			reduced[name] = function
		}
	}

	return reduced
}
