package cli

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
	var pid = int64(event.Pid)

	if pid == 0 {
		return fmt.Errorf("empty pid")
	}

	switch event.Type {
	case EventRequestInit:
		if err := c.handleRequestInit(pid, event); err != nil {
			return fmt.Errorf("failed to process request init: %w", err)
		}
	case EventFunction:
		if err := c.handleFunction(pid, event); err != nil {
			return fmt.Errorf("failed to process function: %w", err)
		}
	case EventRequestShutdown:
		if err := c.handleRequestShutdown(pid, event); err != nil {
			return fmt.Errorf("failed to process request shutdown: %w", err)
		}
	}

	return nil
}

// Process the function event and store the data.
func (c *Handler) handleRequestInit(pid int64, event bpfEvent) error {
	t := trace.Trace{
		Metadata: trace.Metadata{
			Source: trace.SourceCLI,
			ID:     c.getID(pid),
			CLI: trace.MetadataCLI{
				Command: unix.ByteSliceToString(event.Command[:]),
			},
			StartTime: int64(event.Timestamp),
		},
	}

	c.storage.Set(c.getID(pid), t, cache.DefaultExpiration)

	return nil
}

// Process the function event and store the data.
func (c *Handler) handleFunction(pid int64, event bpfEvent) error {
	function := trace.FunctionCall{
		Name: unix.ByteSliceToString(event.FunctionName[:]),
		// The start time is the event time minus how long it look to execute.
		// The event is triggerd after a the function is called and we have collected the elapsed time.
		StartTime: int64(event.Timestamp - event.Elapsed),
		Elapsed:   int64(event.Elapsed),
		Memory:    int64(event.Memory),
	}

	x, found := c.storage.Get(c.getID(pid))
	if !found {
		return fmt.Errorf("not found in storage")
	}

	t := x.(trace.Trace)

	if t.ResourceUtilisation.MaxMemory < function.Memory {
		t.ResourceUtilisation.MaxMemory = function.Memory
	}

	t.FunctionCalls = append(t.FunctionCalls, function)

	c.storage.Set(c.getID(pid), t, cache.DefaultExpiration)

	return nil
}

// Process the request shutdown event and send the profile to the plugin.
func (c *Handler) handleRequestShutdown(pid int64, event bpfEvent) error {
	x, found := c.storage.Get(c.getID(pid))
	if !found {
		return fmt.Errorf("not found in storage")
	}

	t := x.(trace.Trace)

	t.Metadata.EndTime = int64(event.Timestamp)

	// Cleanup this request after we have processed it.
	defer c.storage.Delete(c.getID(pid))

	if len(t.FunctionCalls) == 0 {
		return fmt.Errorf("no functions found for request with id: %s", c.getID(pid))
	}

	err := c.plugin.ProcessTrace(context.TODO(), t)
	if err != nil {
		return fmt.Errorf("failed to send profile data to plugin: %w", err)
	}

	return nil
}

// Returns an ID derived from the pid for tracking between events.
func (c *Handler) getID(pid int64) string {
	return fmt.Sprintf("%d", pid)
}
