// Package collector implements the collection of PHP telemetry data.
package fpm

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"github.com/cilium/ebpf"
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
func Run(ctx context.Context, plugin sink.Interface, extensionPath string) error {
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
		Expire: time.Minute,
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

	// An event whose request we know nothing about cannot be handled, and that
	// is routine rather than exceptional: attaching part way through a request
	// means its events arrive without the request init that would have opened
	// it. These count how often that happened instead of ending the tracer.
	var (
		eventsSkipped            atomic.Int64
		drupalCacheEventsSkipped atomic.Int64
	)

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

			if err := manager.Handle(event); err != nil {
				eventsSkipped.Add(1)
			}
		}
	})

	// Goroutine that reads Drupal cache events from their own ringbuf.
	g.Go(func() error {
		defer drupalReader.Close()

		var event bpfDrupalCacheEvent

		for {
			record, err := drupalReader.Read()
			if err != nil {
				if errors.Is(err, ringbuf.ErrClosed) {
					return nil
				}

				continue
			}

			if err := binary.Read(bytes.NewBuffer(record.RawSample), binary.LittleEndian, &event); err != nil {
				return logger.WrapError(fmt.Errorf("failed to read drupal cache event: %w", err))
			}

			if err := manager.HandleDrupalCache(event); err != nil {
				drupalCacheEventsSkipped.Add(1)
			}
		}
	})

	// Goroutine that reacts to context cancellation
	g.Go(func() error {
		<-ctx.Done()
		_ = reader.Close()
		_ = drupalReader.Close()
		return nil
	})

	err = g.Wait()

	logger.SetAttr("events_skipped", eventsSkipped.Load())
	logger.SetAttr("drupal_cache_events_skipped", drupalCacheEventsSkipped.Load())

	if err != nil && !errors.Is(err, context.Canceled) {
		return logger.WrapError(err)
	}

	return ctx.Err()
}
