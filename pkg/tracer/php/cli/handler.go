package cli

import (
	"context"
	"fmt"
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
)

// Handler for handling events.
type Handler struct {
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
	// Clock relates the monotonic timestamps the probes emit to the wall clock.
	// The zero value reads the offset from the system.
	Clock clock.Monotonic
}

// NewHandler creates a new handler for processing events and sending profiles to the sink.
func NewHandler(plugin sink.Interface, options Options) (*Handler, error) {
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
		functionCalls: functioncalls.NewLimiter(options.MaxFunctionCalls, functioncalls.RuntimePHPCLI),
	}

	return client, nil
}

// Handle the event and process it.
func (c *Handler) Handle(ctx context.Context, event bpfEvent) error {
	var pid = int64(event.Pid)

	if pid == 0 {
		return fmt.Errorf("%w: empty pid", ingest.ErrInvalidIdentifier)
	}

	switch event.Type {
	case EventRequestInit:
		if err := c.handleRequestInit(pid, event.Command[:], event.Timestamp); err != nil {
			return fmt.Errorf("failed to process request init: %w", err)
		}
	case EventFunction:
		if err := c.handleFunction(pid, event.FunctionName[:], event.Timestamp, event.Elapsed, event.Memory); err != nil {
			return fmt.Errorf("failed to process function: %w", err)
		}
	case EventRequestShutdown:
		if err := c.handleRequestShutdown(ctx, pid, event.Timestamp); err != nil {
			return fmt.Errorf("failed to process request shutdown: %w", err)
		}
	}

	return nil
}

// HandleRequestInit processes one compact CLI request-init record.
func (c *Handler) HandleRequestInit(_ context.Context, event bpfRequestInitEvent) error {
	pid := int64(event.Pid)
	if pid == 0 {
		return fmt.Errorf("%w: empty pid", ingest.ErrInvalidIdentifier)
	}
	if err := c.handleRequestInit(pid, event.Command[:], event.Timestamp); err != nil {
		return fmt.Errorf("failed to process request init: %w", err)
	}
	return nil
}

// HandleFunction processes one compact CLI function record.
func (c *Handler) HandleFunction(_ context.Context, event bpfFunctionEvent) error {
	pid := int64(event.Pid)
	if pid == 0 {
		return fmt.Errorf("%w: empty pid", ingest.ErrInvalidIdentifier)
	}
	if err := c.handleFunction(pid, event.FunctionName[:], event.Timestamp, event.Elapsed, event.Memory); err != nil {
		return fmt.Errorf("failed to process function: %w", err)
	}
	return nil
}

// HandleRequestShutdown processes one compact CLI request-shutdown record.
func (c *Handler) HandleRequestShutdown(ctx context.Context, event bpfRequestShutdownEvent) error {
	pid := int64(event.Pid)
	if pid == 0 {
		return fmt.Errorf("%w: empty pid", ingest.ErrInvalidIdentifier)
	}
	if err := c.handleRequestShutdown(ctx, pid, event.Timestamp); err != nil {
		return fmt.Errorf("failed to process request shutdown: %w", err)
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
func (c *Handler) handleRequestInit(pid int64, command []byte, timestamp uint64) error {
	t := trace.Trace{
		Metadata: trace.Metadata{
			Source:  trace.SourceCLI,
			Runtime: trace.RuntimePHP,
			ID:      c.getID(pid),
			CLI: trace.MetadataCLI{
				Command: unix.ByteSliceToString(command),
			},
			StartTime: c.options.Clock.Time(timestamp),
		},
	}

	c.storage.Set(c.getID(pid), t, cache.DefaultExpiration)

	return nil
}

// Process the function event and store the data.
func (c *Handler) handleFunction(pid int64, functionName []byte, timestamp, elapsed, memory uint64) error {
	x, found := c.storage.Get(c.getID(pid))
	if !found {
		return fmt.Errorf("%w: request %q not found in storage", ingest.ErrRequestNotTracked, c.getID(pid))
	}

	t := x.(trace.Trace)

	c.functionCalls.Add(
		&t,
		functionName,
		// The call started at the event time minus how long it took to execute:
		// the probe fires once the function has returned and its elapsed time
		// has been collected.
		c.offset(t.Metadata, timestamp-elapsed),
		time.Duration(elapsed),
		int64(memory),
	)

	c.storage.Set(c.getID(pid), t, cache.DefaultExpiration)

	return nil
}

// Process the request shutdown event and send the profile to the plugin.
func (c *Handler) handleRequestShutdown(ctx context.Context, pid int64, timestamp uint64) error {
	x, found := c.storage.Get(c.getID(pid))
	if !found {
		return fmt.Errorf("%w: request %q not found in storage", ingest.ErrRequestNotTracked, c.getID(pid))
	}

	t := x.(trace.Trace)

	t.Metadata.EndTime = c.options.Clock.Time(timestamp)

	// Cleanup this request after we have processed it.
	defer c.storage.Delete(c.getID(pid))

	if len(t.FunctionCalls) == 0 {
		return fmt.Errorf("%w: no functions found for request with id: %s", ingest.ErrTraceEmpty, c.getID(pid))
	}

	err := c.plugin.ProcessTrace(ctx, t)
	if err != nil {
		return fmt.Errorf("failed to send profile data to plugin: %w", err)
	}

	return nil
}

// Returns an ID derived from the pid for tracking between events.
func (c *Handler) getID(pid int64) string {
	return fmt.Sprintf("%d", pid)
}
