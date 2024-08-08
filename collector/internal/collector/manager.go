package collector

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/patrickmn/go-cache"
	"golang.org/x/sys/unix"

	"github.com/skpr/compass/collector/internal/tracing"
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

type ManagerOptions struct {
	Expire            time.Duration
	RequestThreshold  int64
	FunctionThreshold int64
}

// NewManager creates a new manager.
func NewManager(logger *slog.Logger, plugin plugin.Interface, options ManagerOptions) (*Manager, error) {
	client := &Manager{
		logger:  logger,
		storage: cache.New(options.Expire, options.Expire),
		plugin:  plugin,
	}

	return client, nil
}

// Handle the event and process it.
func (c *Manager) Handle(event bpfEvent) error {
	var (
		eventType = unix.ByteSliceToString(event.Type[:])
		requestID = unix.ByteSliceToString(event.RequestId[:])
	)

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
		ExecutionTime: event.ExecutionTime,
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
		Namespace: make(map[string]tracing.Summary),
		Function:  make(map[string]tracing.Summary),
	}

	for _, function := range functions {
		if function.Name == FunctionNameRoot {
			profile.ExecutionTime = function.ExecutionTime
			continue
		}

		// Add to the namespace group summary.
		namespace := getNamespaceKey(function.Name)

		if namespace != "" {
			n := tracing.Summary{
				TotalExecutionTime: function.ExecutionTime,
				Invocations:        1,
			}

			if _, ok := profile.Namespace[namespace]; ok {
				n.TotalExecutionTime = n.TotalExecutionTime + profile.Namespace[namespace].TotalExecutionTime
				n.Invocations = n.Invocations + profile.Namespace[namespace].Invocations
			}

			profile.Namespace[namespace] = n
		}

		// Only send function data if the execution time is greater than the threshold.
		if profile.ExecutionTime > uint64(c.options.RequestThreshold) {
			f := tracing.Summary{
				TotalExecutionTime: function.ExecutionTime,
				Invocations:        1,
			}

			// Add to the function summary.
			if _, ok := profile.Function[function.Name]; ok {
				f.TotalExecutionTime = f.TotalExecutionTime + profile.Function[function.Name].TotalExecutionTime
				f.Invocations = f.Invocations + profile.Function[function.Name].Invocations
			}

			profile.Function[function.Name] = f
		}
	}

	c.logger.Debug("request event has been processed", "functions", len(profile.Function))

	// Only send function data if the execution time is greater than the threshold.
	// We do a catch all here to ensure we didn't miss anything before.
	if profile.ExecutionTime < uint64(c.options.RequestThreshold) {
		profile.Function = nil
	}

	// Reduce the functions based on threshold.
	profile.Function = reduceFunctions(profile.Function, c.options.FunctionThreshold)

	err := c.plugin.ProcessProfile(profile)
	if err != nil {
		return fmt.Errorf("failed to send profile data to plugin: %w", err)
	}

	return nil
}

// Helper function to reduce the profile output for stdout and cut out unnecessary noise.
func reduceFunctions(functions map[string]tracing.Summary, threshold int64) map[string]tracing.Summary {
	for name, function := range functions {
		if function.TotalExecutionTime < uint64(threshold) {
			delete(functions, name)
		}
	}

	return functions
}

// Return identifier for the namespace.
func getNamespaceKey(namespace string) string {
	sl := strings.Split(namespace, "\\")

	if len(sl) == 0 {
		return ""
	}

	return sl[0]
}
