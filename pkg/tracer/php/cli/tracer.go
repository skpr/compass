// Package collector implements the collection of PHP telemetry data.
package cli

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/ringbuf"
	"github.com/cilium/ebpf/rlimit"
	"github.com/skpr/yolog"
	"golang.org/x/sync/errgroup"

	"github.com/skpr/compass/pkg/php/extension/usdt"
	"github.com/skpr/compass/pkg/tracer/sink"
)

const (
	// LoggerStream is used to identify the logger stream for this tracer.
	LoggerStream = "php-cli-tracer"

	// ProbeProvider is the provider name for the probes.
	ProbeProvider = "compass"

	// ProbeNameRequestInit is the name of the request initialisation probe.
	ProbeNameRequestInit = "cli_request_init"
	// ProbeNameRequestShutdown is the name of the request shutdown probe.
	ProbeNameRequestShutdown = "cli_request_shutdown"
	// ProbeNameFunction is the name of the function probe.
	ProbeNameFunction = "cli_function"

	// ProbeNameCanary is used to enable all the above probes.
	ProbeNameCanary = "canary"

	// ProbeRequestInitArg0 is the name of the BPF variable for the request_init arg0 offset.
	ProbeRequestInitArg0 = "cli_request_init_arg0_offset"
	// ProbeRequestInitArg1 is the name of the BPF variable for the request_init arg1 offset.
	ProbeRequestInitArg1 = "cli_request_init_arg1_offset"
	// ProbeRequestInitArg2 is the name of the BPF variable for the request_init arg2 offset.
	ProbeRequestInitArg2 = "cli_request_init_arg2_offset"

	// ProbeFunctionArg0 is the name of the BPF variable for the php_function arg0 offset.
	ProbeFunctionArg0 = "cli_function_arg0_offset"
	// ProbeFunctionArg1 is the name of the BPF variable for the php_function arg1 offset.
	ProbeFunctionArg1 = "cli_function_arg1_offset"
	// ProbeFunctionArg2 is the name of the BPF variable for the php_function arg2 offset.
	ProbeFunctionArg2 = "cli_function_arg2_offset"
	// ProbeFunctionArg3 is the name of the BPF variable for the php_function arg3 offset.
	ProbeFunctionArg3 = "cli_function_arg3_offset"

	// ProbeRequestShutdownArg0 is the name of the BPF variable for the request_shutdown arg0 offset.
	ProbeRequestShutdownArg0 = "cli_request_shutdown_arg0_offset"
)

// Run the collector.
func Run(ctx context.Context, plugin sink.Interface, extentionPath string) error {
	logger := yolog.NewLogger(LoggerStream)
	defer logger.Log(os.Stdout)

	// Allow the current process to lock memory for eBPF resources.
	if err := rlimit.RemoveMemlock(); err != nil {
		return fmt.Errorf("failed to remove memlock rlimit: %w", err)
	}

	logger.SetAttr("removed_memlock", true)

	// Load the compiled BPF collection spec (does not load into kernel yet).
	spec, err := loadBpf()
	if err != nil {
		return logger.WrapError(err)
	}

	// Parse argument register offsets from the .note.stapsdt section of the
	// extension binary. The Rust compiler may allocate USDT probe arguments to
	// different registers across versions, so we discover the actual offsets at
	// runtime rather than hardcoding them in the BPF C source.
	probeArgs, err := usdt.GetProbeArgs(extentionPath, ProbeProvider, []string{
		ProbeNameRequestInit,
		ProbeNameFunction,
		ProbeNameRequestShutdown,
	})
	if err != nil {
		return logger.WrapError(err)
	}

	logger.SetAttr("probes", []string{ProbeNameCanary, ProbeNameRequestInit, ProbeNameFunction, ProbeNameRequestShutdown})

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
	if len(riArgs) < 2 {
		return logger.WrapError(fmt.Errorf("expected 2 args for %s, got %d", ProbeNameRequestInit, len(riArgs)))
	}

	if err := setVar(ProbeRequestInitArg0, riArgs[0].Offset); err != nil {
		return logger.WrapError(err)
	}

	logger.SetAttr(ProbeRequestInitArg0, riArgs[0].Offset)

	if err := setVar(ProbeRequestInitArg1, riArgs[1].Offset); err != nil {
		return logger.WrapError(err)
	}

	logger.SetAttr(ProbeRequestInitArg1, riArgs[1].Offset)

	// Inject php_function argument offsets.
	// Arg order in the Rust probe: pid, function_name, elapsed, memory.
	fnArgs := probeArgs[ProbeNameFunction]
	if len(fnArgs) < 4 {
		return logger.WrapError(fmt.Errorf("expected 4 args for %s, got %d", ProbeNameFunction, len(fnArgs)))
	}

	if err := setVar(ProbeFunctionArg0, fnArgs[0].Offset); err != nil {
		return logger.WrapError(err)
	}

	logger.SetAttr(ProbeFunctionArg0, fnArgs[0].Offset)

	if err := setVar(ProbeFunctionArg1, fnArgs[1].Offset); err != nil {
		return logger.WrapError(err)
	}

	logger.SetAttr(ProbeFunctionArg1, fnArgs[1].Offset)

	if err := setVar(ProbeFunctionArg2, fnArgs[2].Offset); err != nil {
		return logger.WrapError(err)
	}

	logger.SetAttr(ProbeFunctionArg2, fnArgs[2].Offset)

	if err := setVar(ProbeFunctionArg3, fnArgs[3].Offset); err != nil {
		return logger.WrapError(err)
	}

	logger.SetAttr(ProbeFunctionArg3, fnArgs[3].Offset)

	// Inject request_shutdown argument offsets.
	// Arg order in the Rust probe: request_id.
	rsArgs := probeArgs[ProbeNameRequestShutdown]
	if len(rsArgs) < 1 {
		return logger.WrapError(fmt.Errorf("expected 1 arg for %s, got %d", ProbeNameRequestShutdown, len(rsArgs)))
	}

	if err := setVar(ProbeRequestShutdownArg0, rsArgs[0].Offset); err != nil {
		return logger.WrapError(err)
	}

	logger.SetAttr(ProbeRequestShutdownArg0, rsArgs[0].Offset)

	// Load the BPF programs and maps into the kernel with the rewritten constants.
	objs := bpfObjects{}
	if err := spec.LoadAndAssign(&objs, nil); err != nil {
		return logger.WrapError(fmt.Errorf("failed to load objects: %w", err))
	}
	defer objs.Close()

	ex, err := link.OpenExecutable(extentionPath)
	if err != nil {
		return logger.WrapError(fmt.Errorf("failed to open executable: %w", err))
	}

	logger.SetAttr("executable_path", extentionPath)

	probeCanary, err := usdt.AttachProbe(ex, extentionPath, ProbeProvider, ProbeNameCanary, objs.UprobeCompassCanary)
	if err != nil {
		return logger.WrapError(fmt.Errorf("failed to attach probe: %s: %w", ProbeNameCanary, err))
	}
	defer probeCanary.Close()

	logger.SetAttr(fmt.Sprintf("%s_attached", ProbeNameCanary), true)

	probeRequestInit, err := usdt.AttachProbe(ex, extentionPath, ProbeProvider, ProbeNameRequestInit, objs.UprobeCompassCliRequestInit)
	if err != nil {
		return logger.WrapError(fmt.Errorf("failed to attach probe: %s: %w", ProbeNameRequestInit, err))
	}
	defer probeRequestInit.Close()

	logger.SetAttr(fmt.Sprintf("%s_attached", ProbeNameRequestInit), true)

	probeFunction, err := usdt.AttachProbe(ex, extentionPath, ProbeProvider, ProbeNameFunction, objs.UprobeCompassCliFunction)
	if err != nil {
		return logger.WrapError(fmt.Errorf("failed to attach probe: %s: %w", ProbeNameFunction, err))
	}
	defer probeFunction.Close()

	logger.SetAttr(fmt.Sprintf("%s_attached", ProbeNameFunction), true)

	probeRequest, err := usdt.AttachProbe(ex, extentionPath, ProbeProvider, ProbeNameRequestShutdown, objs.UprobeCompassCliRequestShutdown)
	if err != nil {
		return logger.WrapError(fmt.Errorf("failed to attach probe: %s: %w", ProbeNameRequestShutdown, err))
	}
	defer probeRequest.Close()

	logger.SetAttr(fmt.Sprintf("%s_attached", ProbeNameRequestShutdown), true)

	manager, err := NewHandler(plugin, Options{
		Expire: time.Minute,
	})
	if err != nil {
		return logger.WrapError(fmt.Errorf("unable to initialize event manager: %w", err))
	}

	reader, err := ringbuf.NewReader(objs.Events)
	if err != nil {
		return logger.WrapError(fmt.Errorf("failed to start perf event reader: %w", err))
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
					return nil
				}

				continue
			}

			if err := binary.Read(bytes.NewBuffer(record.RawSample), binary.LittleEndian, &event); err != nil {
				return logger.WrapError(fmt.Errorf("failed to read event: %w", err))
			}

			if err := manager.Handle(ctx, event); err != nil {
				return logger.WrapError(fmt.Errorf("failed to handle event: %w", err))
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
		return logger.WrapError(err)
	}

	return ctx.Err()
}
