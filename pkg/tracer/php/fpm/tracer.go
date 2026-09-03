// Package collector implements the collection of PHP telemetry data.
package fpm

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/ringbuf"
	"github.com/cilium/ebpf/rlimit"
	"github.com/skpr/yolog"
	"golang.org/x/sync/errgroup"

	"github.com/skpr/compass/pkg/php/extension/usdt"
	"github.com/skpr/compass/pkg/tracer/ingest"
	"github.com/skpr/compass/pkg/tracer/ringloss"
	"github.com/skpr/compass/pkg/tracer/ringreader"
	"github.com/skpr/compass/pkg/tracer/sink"
)

const (
	// LoggerStream is used to identify the logger stream for this tracer.
	LoggerStream = "php-fpm-tracer"

	// ProbeProvider is the provider name for the probes.
	ProbeProvider = "compass"

	// ProbeNameRequestInit is the name of the request initialisation probe.
	ProbeNameRequestInit = "fpm_request_init"
	// ProbeNameRequestShutdown is the name of the request shutdown probe.
	ProbeNameRequestShutdown = "fpm_request_shutdown"
	// ProbeNameFunction is the name of the function probe.
	ProbeNameFunction = "fpm_function"

	// ProbeNameDrupalCacheRenderArray is the name of the probe for cacheability derived from a render array.
	ProbeNameDrupalCacheRenderArray = "drupal_cacheablemetadata_createfromrenderarray"
	// ProbeNameDrupalCacheObject is the name of the probe for cacheability derived from an object.
	ProbeNameDrupalCacheObject = "drupal_cacheablemetadata_createfromobject"

	// ProbeNameCanary is used to enable all the above probes.
	ProbeNameCanary = "canary"

	// ProbeRequestInitArg0 is the name of the BPF variable for the request_init arg0 offset.
	ProbeRequestInitArg0 = "fpm_request_init_arg0_offset"
	// ProbeRequestInitArg1 is the name of the BPF variable for the request_init arg1 offset.
	ProbeRequestInitArg1 = "fpm_request_init_arg1_offset"
	// ProbeRequestInitArg2 is the name of the BPF variable for the request_init arg2 offset.
	ProbeRequestInitArg2 = "fpm_request_init_arg2_offset"

	// ProbeFunctionArg0 is the name of the BPF variable for the fpm_function arg0 offset.
	ProbeFunctionArg0 = "fpm_function_arg0_offset"
	// ProbeFunctionArg1 is the name of the BPF variable for the fpm_function arg1 offset.
	ProbeFunctionArg1 = "fpm_function_arg1_offset"
	// ProbeFunctionArg2 is the name of the BPF variable for the fpm_function arg2 offset.
	ProbeFunctionArg2 = "fpm_function_arg2_offset"
	// ProbeFunctionArg3 is the name of the BPF variable for the fpm_function arg3 offset.
	ProbeFunctionArg3 = "fpm_function_arg3_offset"

	// ProbeRequestShutdownArg0 is the name of the BPF variable for the request_shutdown arg0 offset.
	ProbeRequestShutdownArg0 = "fpm_request_shutdown_arg0_offset"
)

// Names of the BPF variables holding the argument offsets for the Drupal cache
// probes, in the order the Rust probes declare them.
var (
	// ProbeDrupalCacheRenderArrayArgs are the offsets for request_id, caller,
	// max_age, tags and contexts.
	ProbeDrupalCacheRenderArrayArgs = []string{
		"drupal_cache_render_array_arg0_offset",
		"drupal_cache_render_array_arg1_offset",
		"drupal_cache_render_array_arg2_offset",
		"drupal_cache_render_array_arg3_offset",
		"drupal_cache_render_array_arg4_offset",
	}

	// ProbeDrupalCacheObjectArgs are the offsets for request_id, caller,
	// max_age, object_type, tags and contexts.
	ProbeDrupalCacheObjectArgs = []string{
		"drupal_cache_object_arg0_offset",
		"drupal_cache_object_arg1_offset",
		"drupal_cache_object_arg2_offset",
		"drupal_cache_object_arg3_offset",
		"drupal_cache_object_arg4_offset",
		"drupal_cache_object_arg5_offset",
	}
)

// Run the collector.
func Run(ctx context.Context, plugin sink.Interface, extensionPath string, maxFunctionCalls int) error {
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
	probeArgs, err := usdt.GetProbeArgs(extensionPath, ProbeProvider, []string{
		ProbeNameRequestInit,
		ProbeNameFunction,
		ProbeNameRequestShutdown,
	})
	if err != nil {
		return logger.WrapError(err)
	}

	logger.SetAttr("probes", []string{ProbeNameCanary, ProbeNameRequestInit, ProbeNameFunction, ProbeNameRequestShutdown})

	// The Drupal probes were added to the extension after the ones above, so an
	// older extension will not have them. Losing Drupal cacheability is a much
	// better outcome than losing all PHP tracing, so they are looked up
	// separately and skipped when they are absent.
	drupalArgs, drupalMissing, err := usdt.GetProbeArgsOptional(extensionPath, ProbeProvider, []string{
		ProbeNameDrupalCacheRenderArray,
		ProbeNameDrupalCacheObject,
	})
	if err != nil {
		return logger.WrapError(err)
	}

	if len(drupalMissing) > 0 {
		logger.SetAttr("probes_missing", drupalMissing)
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

	// Inject php_function argument offsets.
	// Arg order in the Rust probe: request_id, function_name, elapsed, memory.
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

	// Inject the Drupal cache probe argument offsets, for whichever of them this
	// extension carries.
	for probe, names := range map[string][]string{
		ProbeNameDrupalCacheRenderArray: ProbeDrupalCacheRenderArrayArgs,
		ProbeNameDrupalCacheObject:      ProbeDrupalCacheObjectArgs,
	} {
		args, ok := drupalArgs[probe]
		if !ok {
			continue
		}

		if len(args) < len(names) {
			return logger.WrapError(fmt.Errorf("expected %d args for %s, got %d", len(names), probe, len(args)))
		}

		for i, name := range names {
			if err := setVar(name, args[i].Offset); err != nil {
				return logger.WrapError(err)
			}

			logger.SetAttr(name, args[i].Offset)
		}
	}

	// Load the BPF programs and maps into the kernel with the rewritten constants.
	objs := bpfObjects{}
	if err := spec.LoadAndAssign(&objs, nil); err != nil {
		return logger.WrapError(fmt.Errorf("failed to load objects: %w", err))
	}
	defer objs.Close()

	reserveFailures, err := ringloss.NewObserver(
		objs.RingbufReserveFailures,
		ingest.RuntimePHPFPM,
		ringreader.StreamEvents,
		ringreader.StreamDrupalCache,
	)
	if err != nil {
		return logger.WrapError(fmt.Errorf("failed to observe ring-buffer reserve failures: %w", err))
	}
	defer func() {
		if err := reserveFailures.Observe(); err != nil {
			logger.AddError(err)
		}
		logger.SetAttr("ringbuf_reserve_failures", reserveFailures.Total(ringreader.StreamEvents))
		logger.SetAttr("drupal_cache_ringbuf_reserve_failures", reserveFailures.Total(ringreader.StreamDrupalCache))
	}()

	ex, err := link.OpenExecutable(extensionPath)
	if err != nil {
		return logger.WrapError(fmt.Errorf("failed to open executable: %w", err))
	}

	logger.SetAttr("executable_path", extensionPath)

	probeCanary, err := usdt.AttachProbe(ex, extensionPath, ProbeProvider, ProbeNameCanary, objs.UprobeCompassCanary)
	if err != nil {
		return logger.WrapError(fmt.Errorf("failed to attach probe: %s: %w", ProbeNameCanary, err))
	}
	defer probeCanary.Close()

	logger.SetAttr(fmt.Sprintf("%s_attached", ProbeNameCanary), true)

	probeRequestInit, err := usdt.AttachProbe(ex, extensionPath, ProbeProvider, ProbeNameRequestInit, objs.UprobeCompassFpmRequestInit)
	if err != nil {
		return logger.WrapError(fmt.Errorf("failed to attach probe: %s: %w", ProbeNameRequestInit, err))
	}
	defer probeRequestInit.Close()

	logger.SetAttr(fmt.Sprintf("%s_attached", ProbeNameRequestInit), true)

	probeFunction, err := usdt.AttachProbe(ex, extensionPath, ProbeProvider, ProbeNameFunction, objs.UprobeCompassFpmFunction)
	if err != nil {
		return logger.WrapError(fmt.Errorf("failed to attach probe: %s: %w", ProbeNameFunction, err))
	}
	defer probeFunction.Close()

	logger.SetAttr(fmt.Sprintf("%s_attached", ProbeNameFunction), true)

	probeRequestShutdown, err := usdt.AttachProbe(ex, extensionPath, ProbeProvider, ProbeNameRequestShutdown, objs.UprobeCompassFpmRequestShutdown)
	if err != nil {
		return logger.WrapError(fmt.Errorf("failed to attach probe: %s: %w", ProbeNameRequestShutdown, err))
	}
	defer probeRequestShutdown.Close()

	logger.SetAttr(fmt.Sprintf("%s_attached", ProbeNameRequestShutdown), true)

	for probe, prog := range map[string]*ebpf.Program{
		ProbeNameDrupalCacheRenderArray: objs.UprobeCompassDrupalCacheRenderArray,
		ProbeNameDrupalCacheObject:      objs.UprobeCompassDrupalCacheObject,
	} {
		if _, ok := drupalArgs[probe]; !ok {
			continue
		}

		attached, err := usdt.AttachProbe(ex, extensionPath, ProbeProvider, probe, prog)
		if err != nil {
			return logger.WrapError(fmt.Errorf("failed to attach probe: %s: %w", probe, err))
		}
		defer attached.Close()

		logger.SetAttr(fmt.Sprintf("%s_attached", probe), true)
	}

	manager, err := NewHandler(plugin, Options{
		Expire:           time.Minute,
		MaxFunctionCalls: maxFunctionCalls,
	})
	if err != nil {
		return logger.WrapError(fmt.Errorf("unable to initialize event manager: %w", err))
	}

	reader, err := ringbuf.NewReader(objs.Events)
	if err != nil {
		return logger.WrapError(fmt.Errorf("failed to start perf event reader: %w", err))
	}

	drupalReader, err := ringbuf.NewReader(objs.DrupalCacheEvents)
	if err != nil {
		return logger.WrapError(fmt.Errorf("failed to start drupal cache event reader: %w", err))
	}

	eventsSkipped := ingest.NewSkips(ingest.RuntimePHPFPM)
	drupalCacheEventsSkipped := ingest.NewSkips(ingest.RuntimePHPFPM)

	group, runCtx := errgroup.WithContext(ctx)
	group.Go(func() error {
		return reserveFailures.Run(runCtx)
	})
	group.Go(func() error {
		return ringreader.Run(runCtx,
			ringreader.Source{
				Reader:  reader,
				Runtime: ingest.RuntimePHPFPM,
				Stream:  ringreader.StreamEvents,
				Handle: func(readCtx context.Context, rawSample []byte) error {
					return processEvent(readCtx, rawSample, manager, eventsSkipped)
				},
			},
			ringreader.Source{
				Reader:  drupalReader,
				Runtime: ingest.RuntimePHPFPM,
				Stream:  ringreader.StreamDrupalCache,
				Handle: func(readCtx context.Context, rawSample []byte) error {
					if err := processDrupalCacheEvent(readCtx, rawSample, manager, drupalCacheEventsSkipped); err != nil {
						return fmt.Errorf("failed to process drupal cache event: %w", err)
					}
					return nil
				},
			},
		)
	})

	err = group.Wait()

	logger.SetAttr("events_skipped", eventsSkipped.Total())
	logger.SetAttr("events_skipped_request_not_tracked", eventsSkipped.Count(ingest.ReasonRequestNotTracked))
	logger.SetAttr("events_skipped_invalid_identifier", eventsSkipped.Count(ingest.ReasonInvalidIdentifier))
	logger.SetAttr("events_skipped_trace_empty", eventsSkipped.Count(ingest.ReasonTraceEmpty))
	logger.SetAttr("drupal_cache_events_skipped", drupalCacheEventsSkipped.Total())

	if err != nil && !errors.Is(err, context.Canceled) {
		return logger.WrapError(err)
	}

	return ctx.Err()
}

func processEvent(ctx context.Context, rawSample []byte, manager *Handler, skips *ingest.Skips) error {
	return ingest.DecodeAndHandle(ctx, rawSample, manager.Handle, skips)
}

func processDrupalCacheEvent(ctx context.Context, rawSample []byte, manager *Handler, skips *ingest.Skips) error {
	return ingest.DecodeAndHandle(ctx, rawSample, manager.HandleDrupalCache, skips)
}
