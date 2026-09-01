// Package collector implements the collection of node telemetry data.
package http

import (
	"context"
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
	"github.com/skpr/compass/pkg/tracer/ingest"
	"github.com/skpr/compass/pkg/tracer/sink"
)

const (
	// LoggerStream is used to identify the logger stream for this tracer.
	LoggerStream = "node-http-tracer"

	// ProbeProvider is the provider name for the probes.
	ProbeProvider = "compass"

	// ProbeNameRequestInit is the name of the request initialisation probe.
	ProbeNameRequestInit = "http_request_init"
	// ProbeNameRequestShutdown is the name of the request shutdown probe.
	ProbeNameRequestShutdown = "http_request_shutdown"
	// ProbeNameFunction is the name of the function probe.
	ProbeNameFunction = "http_function"

	// ProbeNameCanary is used to enable all the above probes.
	ProbeNameCanary = "canary"

	// ProbeRequestInitArg0 is the name of the BPF variable for the request_init arg0 offset.
	ProbeRequestInitArg0 = "http_request_init_arg0_offset"
	// ProbeRequestInitArg1 is the name of the BPF variable for the request_init arg1 offset.
	ProbeRequestInitArg1 = "http_request_init_arg1_offset"
	// ProbeRequestInitArg2 is the name of the BPF variable for the request_init arg2 offset.
	ProbeRequestInitArg2 = "http_request_init_arg2_offset"

	// ProbeFunctionArg0 is the name of the BPF variable for the http_function arg0 offset.
	ProbeFunctionArg0 = "http_function_arg0_offset"
	// ProbeFunctionArg1 is the name of the BPF variable for the http_function arg1 offset.
	ProbeFunctionArg1 = "http_function_arg1_offset"
	// ProbeFunctionArg2 is the name of the BPF variable for the http_function arg2 offset.
	ProbeFunctionArg2 = "http_function_arg2_offset"
	// ProbeFunctionArg3 is the name of the BPF variable for the http_function arg3 offset.
	ProbeFunctionArg3 = "http_function_arg3_offset"

	// ProbeRequestShutdownArg0 is the name of the BPF variable for the request_shutdown arg0 offset.
	ProbeRequestShutdownArg0 = "http_request_shutdown_arg0_offset"
)

// Run the collector.
func Run(ctx context.Context, plugin sink.Interface, addonPath string) error {
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
	probeArgs, err := usdt.GetProbeArgs(addonPath, ProbeProvider, []string{
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
	if len(riArgs) < 3 {
		return logger.WrapError(fmt.Errorf("expected 3 args for %s, got %d", ProbeNameRequestInit, len(riArgs)))
	}

	if err := setVar(ProbeRequestInitArg0, riArgs[0].Offset); err != nil {
		return logger.WrapError(err)
	}

	logger.SetAttr(ProbeRequestInitArg0, riArgs[0].Offset)

	if err := setVar(ProbeRequestInitArg1, riArgs[1].Offset); err != nil {
		return logger.WrapError(err)
	}

	logger.SetAttr(ProbeRequestInitArg1, riArgs[1].Offset)

	if err := setVar(ProbeRequestInitArg2, riArgs[2].Offset); err != nil {
		return logger.WrapError(err)
	}

	logger.SetAttr(ProbeRequestInitArg2, riArgs[2].Offset)

	// Inject http_function argument offsets.
	// Arg order in the probe: request_id, function_name, elapsed, memory.
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

	ex, err := link.OpenExecutable(addonPath)
	if err != nil {
		return logger.WrapError(fmt.Errorf("failed to open executable: %w", err))
	}

	logger.SetAttr("executable_path", addonPath)

	probeCanary, err := usdt.AttachProbe(ex, addonPath, ProbeProvider, ProbeNameCanary, objs.UprobeCompassCanary)
	if err != nil {
		return logger.WrapError(fmt.Errorf("failed to attach probe: %s: %w", ProbeNameCanary, err))
	}
	defer probeCanary.Close()

	logger.SetAttr(fmt.Sprintf("%s_attached", ProbeNameCanary), true)

	probeRequestInit, err := usdt.AttachProbe(ex, addonPath, ProbeProvider, ProbeNameRequestInit, objs.UprobeCompassHttpRequestInit)
	if err != nil {
		return logger.WrapError(fmt.Errorf("failed to attach probe: %s: %w", ProbeNameRequestInit, err))
	}
	defer probeRequestInit.Close()

	logger.SetAttr(fmt.Sprintf("%s_attached", ProbeNameRequestInit), true)

	probeFunction, err := usdt.AttachProbe(ex, addonPath, ProbeProvider, ProbeNameFunction, objs.UprobeCompassHttpFunction)
	if err != nil {
		return logger.WrapError(fmt.Errorf("failed to attach probe: %s: %w", ProbeNameFunction, err))
	}
	defer probeFunction.Close()

	logger.SetAttr(fmt.Sprintf("%s_attached", ProbeNameFunction), true)

	probeRequestShutdown, err := usdt.AttachProbe(ex, addonPath, ProbeProvider, ProbeNameRequestShutdown, objs.UprobeCompassHttpRequestShutdown)
	if err != nil {
		return logger.WrapError(fmt.Errorf("failed to attach probe: %s: %w", ProbeNameRequestShutdown, err))
	}
	defer probeRequestShutdown.Close()

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
	skips := ingest.NewSkips(ingest.RuntimeNodeHTTP)

	g, ctx := errgroup.WithContext(ctx)

	// Goroutine that reads from the ringbuf and handles traces.
	g.Go(func() error {
		defer reader.Close()

		for {
			record, err := reader.Read()
			if err != nil {
				// Closed because ctx was cancelled or someone explicitly closed it.
				if errors.Is(err, ringbuf.ErrClosed) {
					return nil
				}

				continue
			}

			if err := processEvent(ctx, record.RawSample, manager, skips); err != nil {
				return logger.WrapError(err)
			}
		}
	})

	// Goroutine that reacts to context cancellation
	g.Go(func() error {
		<-ctx.Done()
		_ = reader.Close()
		return nil
	})

	err = g.Wait()

	logger.SetAttr("events_skipped", skips.Total())
	logger.SetAttr("events_skipped_request_not_tracked", skips.Count(ingest.ReasonRequestNotTracked))
	logger.SetAttr("events_skipped_invalid_identifier", skips.Count(ingest.ReasonInvalidIdentifier))
	logger.SetAttr("events_skipped_trace_empty", skips.Count(ingest.ReasonTraceEmpty))

	if err != nil && !errors.Is(err, context.Canceled) {
		return logger.WrapError(err)
	}

	return ctx.Err()
}

func processEvent(ctx context.Context, rawSample []byte, manager *Handler, skips *ingest.Skips) error {
	return ingest.DecodeAndHandle(ctx, rawSample, manager.Handle, skips)
}
