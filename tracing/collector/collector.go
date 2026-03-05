// Package collector implements the collection of PHP telemetry data.
package collector

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/ringbuf"
	"github.com/cilium/ebpf/rlimit"
	"golang.org/x/sync/errgroup"

	"github.com/skpr/compass/tracing/collector/sink"
	"github.com/skpr/compass/tracing/collector/usdt"
)

const (
	// ProbeProvider is the provider name for the probes.
	ProbeProvider = "compass"

	// ProbeNameRequestInit is the name of the request initialisation probe.
	ProbeNameRequestInit = "request_init"
	// ProbeNameRequestShutdown is the name of the request shutdown probe.
	ProbeNameRequestShutdown = "request_shutdown"
	// ProbeNameFunction is the name of the function probe.
	ProbeNameFunction = "php_function"

	// ProbeNameCanary is used to enable all the above probes.
	ProbeNameCanary = "canary"
)

// RunOptions for configuring the collector.
type RunOptions struct {
	ExecutablePath string
}

// Run the collector.
func Run(ctx context.Context, logger Logger, plugin sink.Interface, options RunOptions) error {
	logger.Info("Loading probes")

	// Allow the current process to lock memory for eBPF resources.
	if err := rlimit.RemoveMemlock(); err != nil {
		return fmt.Errorf("failed to remove memlock rlimit: %w", err)
	}

	// Load the compiled BPF collection spec (does not load into kernel yet).
	spec, err := loadBpf()
	if err != nil {
		return fmt.Errorf("failed to load bpf spec: %w", err)
	}

	logger.Info("Resolving probe argument offsets from USDT notes")

	// Parse argument register offsets from the .note.stapsdt section of the
	// extension binary. The Rust compiler may allocate USDT probe arguments to
	// different registers across versions, so we discover the actual offsets at
	// runtime rather than hardcoding them in the BPF C source.
	probeArgs, err := usdt.GetProbeArgs(options.ExecutablePath, ProbeProvider, []string{
		ProbeNameRequestInit,
		ProbeNameFunction,
		ProbeNameRequestShutdown,
	})
	if err != nil {
		return fmt.Errorf("failed to resolve probe args: %w", err)
	}

	// setVar is a helper that sets a named volatile const in the BPF spec.
	setVar := func(name string, val uint32) error {
		v, ok := spec.Variables[name]
		if !ok {
			return fmt.Errorf("BPF variable %q not found in spec", name)
		}
		return v.Set(val)
	}

	// Inject request_init argument offsets.
	// Arg order in the Rust probe: request_id, uri, method.
	riArgs := probeArgs[ProbeNameRequestInit]
	if len(riArgs) < 3 {
		return fmt.Errorf("expected 3 args for %s, got %d", ProbeNameRequestInit, len(riArgs))
	}
	if err := setVar("request_init_arg0_offset", riArgs[0].Offset); err != nil {
		return err
	}
	if err := setVar("request_init_arg1_offset", riArgs[1].Offset); err != nil {
		return err
	}
	if err := setVar("request_init_arg2_offset", riArgs[2].Offset); err != nil {
		return err
	}

	// Inject php_function argument offsets.
	// Arg order in the Rust probe: request_id, function_name, elapsed.
	fnArgs := probeArgs[ProbeNameFunction]
	if len(fnArgs) < 3 {
		return fmt.Errorf("expected 3 args for %s, got %d", ProbeNameFunction, len(fnArgs))
	}
	if err := setVar("php_function_arg0_offset", fnArgs[0].Offset); err != nil {
		return err
	}
	if err := setVar("php_function_arg1_offset", fnArgs[1].Offset); err != nil {
		return err
	}
	if err := setVar("php_function_arg2_offset", fnArgs[2].Offset); err != nil {
		return err
	}

	// Inject request_shutdown argument offsets.
	// Arg order in the Rust probe: request_id.
	rsArgs := probeArgs[ProbeNameRequestShutdown]
	if len(rsArgs) < 1 {
		return fmt.Errorf("expected 1 arg for %s, got %d", ProbeNameRequestShutdown, len(rsArgs))
	}
	if err := setVar("request_shutdown_arg0_offset", rsArgs[0].Offset); err != nil {
		return err
	}

	// Load the BPF programs and maps into the kernel with the rewritten constants.
	objs := bpfObjects{}
	if err := spec.LoadAndAssign(&objs, nil); err != nil {
		return fmt.Errorf("failed to load objects: %w", err)
	}
	defer objs.Close()

	logger.Info("Opening executable")

	ex, err := link.OpenExecutable(options.ExecutablePath)
	if err != nil {
		return fmt.Errorf("failed to open executable: %w", err)
	}

	logger.Info("Attaching probes")

	probeCanary, err := usdt.AttachProbe(ex, options.ExecutablePath, ProbeProvider, ProbeNameCanary, objs.UprobeCompassCanary)
	if err != nil {
		return fmt.Errorf("failed to attach probe: %s: %w", ProbeNameCanary, err)
	}
	defer probeCanary.Close()

	probeRequestInit, err := usdt.AttachProbe(ex, options.ExecutablePath, ProbeProvider, ProbeNameRequestInit, objs.UprobeCompassRequestInit)
	if err != nil {
		return fmt.Errorf("failed to attach probe: %s: %w", ProbeNameRequestInit, err)
	}
	defer probeRequestInit.Close()

	probeFunction, err := usdt.AttachProbe(ex, options.ExecutablePath, ProbeProvider, ProbeNameFunction, objs.UprobeCompassPhpFunction)
	if err != nil {
		return fmt.Errorf("failed to attach probe: %s: %w", ProbeNameFunction, err)
	}
	defer probeFunction.Close()

	probeRequest, err := usdt.AttachProbe(ex, options.ExecutablePath, ProbeProvider, ProbeNameRequestShutdown, objs.UprobeCompassRequestShutdown)
	if err != nil {
		return fmt.Errorf("failed to attach probe: %s: %w", ProbeNameRequestShutdown, err)
	}
	defer probeRequest.Close()

	logger.Info("Starting event manager..")

	manager, err := NewManager(logger, plugin, Options{
		Expire: time.Minute,
	})
	if err != nil {
		return fmt.Errorf("unable to initialize event manager: %w", err)
	}

	logger.Info("Listening for events..")

	reader, err := ringbuf.NewReader(objs.Events)
	if err != nil {
		return fmt.Errorf("failed to start perf event reader: %w", err)
	}

	g, ctx := errgroup.WithContext(ctx)

	// Goroutine that reads from the ringbuf and handles traces.
	g.Go(func() error {
		defer reader.Close()

		var event bpfEvent

		for {
			record, err := reader.Read()
			if err != nil {
				// Closed because ctx was cancelled or someone explicitly closed it.
				if errors.Is(err, ringbuf.ErrClosed) {
					logger.Info("ringbuf reader closed, exiting read loop")
					return nil
				}

				logger.Error("failed to read from perf event reader", slog.Any("err", err))
				continue
			}

			if err := binary.Read(bytes.NewBuffer(record.RawSample), binary.LittleEndian, &event); err != nil {
				logger.Error("failed to read event", slog.Any("err", err))
				continue
			}

			if err := manager.Handle(event); err != nil {
				logger.Error("failed to handle event", slog.Any("err", err))
				continue
			}
		}
	})

	// Goroutine that reacts to context cancellation
	g.Go(func() error {
		<-ctx.Done()
		_ = reader.Close()
		return nil
	})

	if err := g.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}

	return ctx.Err()
}
