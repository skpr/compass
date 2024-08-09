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

	"github.com/skpr/compass/collector/internal/usdt"
	"github.com/skpr/compass/collector/plugin"
)

const (
	// ProbeProvider is the provider name for the probes.
	ProbeProvider = "compass"
	// ProbeNameRequestShutdown is the name of the request shutdown probe.
	ProbeNameRequestShutdown = "request_shutdown"
	// ProbeNameFunction is the name of the function probe.
	ProbeNameFunction = "php_function"
)

//go:generate bpf2go -target amd64 -type event bpf program.bpf.c -- -I./headers

type RunOptions struct {
	ExecutablePath    string
	RequestThreshold  float64
	FunctionThreshold float64
}

// Run the collector.
func Run(ctx context.Context, logger *slog.Logger, plugin plugin.Interface, options RunOptions) error {
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

	ex, err := link.OpenExecutable(options.ExecutablePath)
	if err != nil {
		return fmt.Errorf("failed to open executable: %w", err)
	}

	logger.Info("Attaching probes")

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

	manager, err := NewManager(logger, plugin, ManagerOptions{
		Expire:            time.Minute,
		RequestThreshold:  options.RequestThreshold,
		FunctionThreshold: options.FunctionThreshold,
	})
	if err != nil {
		return fmt.Errorf("unable to initialize event manager: %w", err)
	}

	logger.Info("Listening for events..")

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
				if errors.Is(err, ringbuf.ErrClosed) {
					return err
				}

				logger.Info("reading from perf event reader:", err)
				continue
			}

			// Parse the event entry into a bpfEvent structure.
			if err := binary.Read(bytes.NewBuffer(record.RawSample), binary.LittleEndian, &event); err != nil {
				return fmt.Errorf("failed to parse event: %w", err)
			}

			if err := manager.Handle(event); err != nil {
				return fmt.Errorf("failed to handle event: %w", err)
			}
		}
	}
}
