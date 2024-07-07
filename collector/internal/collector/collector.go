package collector

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"time"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/perf"
	"github.com/cilium/ebpf/ringbuf"
	"github.com/cilium/ebpf/rlimit"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sys/unix"

	"github.com/skpr/compass/collector/internal/event/manager"
	"github.com/skpr/compass/collector/internal/event/types"
	"github.com/skpr/compass/collector/internal/usdt"
	"github.com/skpr/compass/collector/plugin"
)

const (
	ProbeProvider            = "compass"
	ProbeNameRequestInit     = "fpm_request_init"
	ProbeNameRequestShutdown = "fpm_request_shutdown"
	ProbeNameFunctionBegin   = "php_function_begin"
	ProbeNameFunctionEnd     = "php_function_end"
)

//go:generate bpf2go -target amd64 -type event bpf program.bpf.c -- -I./headers

func Run(ctx context.Context, logger *slog.Logger, executablePath string, plugin plugin.Interface, debug bool) error {
	logger.Info("Loading probes")

	// Allow the current process to lock memory for eBPF resources.
	if err := rlimit.RemoveMemlock(); err != nil {
		return fmt.Errorf("failed to remove memlock rlimit: %w", err)
	}

	// Load pre-compiled programs and maps into the kernel.
	objs := bpfObjects{}
	if err := loadBpfObjects(&objs, nil); err != nil {
		return fmt.Errorf("failed to load objects: %w", err)
	}
	defer objs.Close()

	logger.Info("Opening executable")

	ex, err := link.OpenExecutable(executablePath)
	if err != nil {
		return fmt.Errorf("failed to open executable: %w", err)
	}

	logger.Info("Attaching probes")

	probeRequestInit, err := attachProbe(ex, executablePath, ProbeProvider, ProbeNameRequestInit, objs.UprobeCompassFpmRequestInit)
	if err != nil {
		return fmt.Errorf("failed to attach probe: %s: %w", ProbeNameRequestInit, err)
	}
	defer probeRequestInit.Close()

	probeRequestShutdown, err := attachProbe(ex, executablePath, ProbeProvider, ProbeNameRequestShutdown, objs.UprobeCompassFpmRequestShutdown)
	if err != nil {
		return fmt.Errorf("failed to attach probe: %s: %w", ProbeNameRequestShutdown, err)
	}
	defer probeRequestShutdown.Close()

	probeFunctionBegin, err := attachProbe(ex, executablePath, ProbeProvider, ProbeNameFunctionBegin, objs.UprobeCompassPhpFunctionBegin)
	if err != nil {
		return fmt.Errorf("failed to attach probe: %s: %w", ProbeNameFunctionBegin, err)
	}
	defer probeFunctionBegin.Close()

	probeFunctionEnd, err := attachProbe(ex, executablePath, ProbeProvider, ProbeNameFunctionEnd, objs.UprobeCompassPhpFunctionEnd)
	if err != nil {
		return fmt.Errorf("failed to attach probe: %s: %w", ProbeNameFunctionEnd, err)
	}
	defer probeFunctionEnd.Close()

	logger.Info("Starting event mangaer..")

	manager, err := manager.New()
	if err != nil {
		return fmt.Errorf("unable to initialize event manager: %w", err)
	}

	logger.Info("Listening for events..")

	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		return manager.RunWithExpiration(ctx, time.Minute*5)
	})

	eg.Go(func() error {
		reader, err := ringbuf.NewReader(objs.Events)
		if err != nil {
			return fmt.Errorf("failed to start perf event reader: %w", err)
		}
		defer reader.Close()

		// bpfEvent is generated by bpf2go.
		var event bpfEvent

		for {
			select {
			case <-ctx.Done():
				return reader.Close()
			default:
				record, err := reader.Read()
				if err != nil {
					if errors.Is(err, perf.ErrClosed) {
						return err
					}

					logger.Info("reading from perf event reader:", err)
					continue
				}

				// Parse the perf event entry into a bpfEvent structure.
				if err := binary.Read(bytes.NewBuffer(record.RawSample), binary.LittleEndian, &event); err != nil {
					return fmt.Errorf("failed to parse event: %w", err)
				}

				var (
					eventType     = unix.ByteSliceToString(event.Type[:])
					requestID     = unix.ByteSliceToString(event.RequestId[:])
					name          = unix.ByteSliceToString(event.Name[:])
					executionTime = event.ExecutionTime
				)

				if eventType == "function" {
					if debug {
						logger.Debug("function event has been called", "request_id", requestID, "name", name, "execution_time", executionTime)
					}

					err = manager.AddFunction(requestID, name, executionTime, time.Minute)
					if err != nil {
						log.Printf("failed to add function to event manager: %s", err)
					}

					continue
				}

				if eventType == "request" {
					if debug {
						logger.Debug("request event has been called", "request_id", requestID, "execution_time", executionTime)
					}

					functions, err := manager.FlushRequest(requestID)
					if err != nil {
						log.Printf("failed to add function to event manager: %s", err)
						continue
					}

					if debug {
						logger.Debug("request event has %d functions associated", len(functions))
					}

					trace := types.Trace{
						ID:                 requestID,
						TotalExecutionTime: event.ExecutionTime,
						Functions:          functions,
					}

					err = plugin.TraceEnd(trace)
					if err != nil {
						return fmt.Errorf("failed to send profile data to plugin: %w", err)
					}

					continue
				}
			}
		}
	})

	return eg.Wait()
}

// Helper function to lookup the (usdt) probes location and attach our eBPF program to it.
func attachProbe(ex *link.Executable, executable, provider, probe string, prog *ebpf.Program) (link.Link, error) {
	locationFunctionBegin, err := usdt.GetLocationFromProbe(executable, provider, probe)
	if err != nil {
		return nil, fmt.Errorf("failed to get probe location: %s: %w", probe, err)
	}

	return ex.Uprobe(getSymbol(provider, probe), prog, &link.UprobeOptions{
		Address:      locationFunctionBegin.Location,
		RefCtrOffset: locationFunctionBegin.SemaphoreOffsetRefctr,
	})
}

func getSymbol(provider, function string) string {
	return fmt.Sprintf("usdt_%s_%s", provider, function)
}
