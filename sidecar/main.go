// Package main provides the entrypoint for the sidecar.
package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/christgf/env"
	"github.com/spf13/cobra"

	"github.com/skpr/compass/collector"
	"github.com/skpr/compass/collector/extension/discovery"
	"github.com/skpr/compass/collector/sink"
	"github.com/skpr/compass/sidecar/sink/otel"
	"github.com/skpr/compass/sidecar/sink/stdout"
)

var (
	cmdLong = `
		Run the sidecar which collects profiles and prints them to stdout.`

	cmdExample = `
		# Run the sidecar with the defaults.
		compass-sidecar

		# Run the sidecar with all the thresholds disabled.
		export COMPASS_FUNCTION_THRESHOLD=0
		export COMPASS_REQUEST_THRESHOLD=0
		compass-sidecar

		# Enable debugging.
		export COMPASS_LOG_LEVEL=info
		compass-sidecar`
)

// Options for this sidecar application.
type Options struct {
	ProcessName       string
	ExtensionPath     string
	LogLevel          string
	Sink              string
	FunctionThreshold int64
	RequestThreshold  int64
	OtelServiceName   string
	OtelEndpoint      string
}

func main() {
	o := Options{}

	cmd := &cobra.Command{
		Use:     "compass-sidecar",
		Short:   "Run the Compass sidecar",
		Long:    cmdLong,
		Example: cmdExample,
		RunE: func(cmd *cobra.Command, _ []string) error {
			lvl := new(slog.LevelVar)

			switch o.LogLevel {
			case "info":
				lvl.Set(slog.LevelInfo)
			case "debug":
				lvl.Set(slog.LevelDebug)
			case "warn":
				lvl.Set(slog.LevelWarn)
			case "error":
				lvl.Set(slog.LevelError)
			default:
				lvl.Set(slog.LevelInfo)
			}

			logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
				Level: lvl,
			}))

			logger.Info("Looking for extension", "process_name", o.ProcessName)

			path, err := discovery.GetPathFromProcess(logger, o.ProcessName, o.ExtensionPath)
			if err != nil {
				return err
			}

			logger.Info("Extension found", "process_name", o.ProcessName, "extension_path", path)

			logger.Info("Starting collector")

			sink, err := o.getSink(logger)
			if err != nil {
				return fmt.Errorf("failed to get sink: %w", err)
			}

			err = collector.Run(cmd.Context(), logger, sink, collector.RunOptions{
				ExecutablePath: path,
			})
			if err != nil {
				return fmt.Errorf("collector failed: %w", err)
			}

			logger.Info("Collector finished")

			return nil
		},
	}

	// Extension discovery flags.
	cmd.PersistentFlags().StringVar(&o.ProcessName, "process-name", env.String("COMPASS_PROCESS_NAME", "php-fpm"), "Name of the process which will be used for discovery")
	cmd.PersistentFlags().StringVar(&o.ExtensionPath, "extension-path", env.String("COMPASS_EXTENSION_PATH", "/usr/lib/php/modules/compass.so"), "Path to the Compass extension")

	// Sink configuration.
	cmd.PersistentFlags().StringVar(&o.Sink, "sink", env.String("COMPASS_SIDECAR_SINK", "stdout"), "Which sink to use for tracing data")
	cmd.PersistentFlags().Int64Var(&o.FunctionThreshold, "function-threshold", env.Int64("COMPASS_SIDECAR_FUNCTION_THRESHOLD", 10000), "Watermark for which functionss to trace")
	cmd.PersistentFlags().Int64Var(&o.RequestThreshold, "request-threshold", env.Int64("COMPASS_SIDECAR_REQUEST_THRESHOLD", 100000), "Watermark for which requests to trace")

	// Debugging.
	cmd.PersistentFlags().StringVar(&o.LogLevel, "log-level", env.String("COMPASS_SIDECAR_LOG_LEVEL", "info"), "Set the logging level")

	// OpenTelemetry.
	cmd.PersistentFlags().StringVar(&o.OtelServiceName, "otel-service-name", env.String("COMPASS_SIDECAR_OTEL_SERVICE_NAME", ""), "Configure the service name that will be associated in OpenTelemetry")
	cmd.PersistentFlags().StringVar(&o.OtelEndpoint, "otel-endpoint", env.String("COMPASS_SIDECAR_OTEL_ENDPOINT", "http://jaeger:4318/v1/traces"), "Configure where OpenTelemetry traces are sent")

	err := cmd.Execute()
	if err != nil {
		panic(err)
	}
}

func (o Options) getSink(logger *slog.Logger) (sink.Interface, error) {
	switch o.Sink {
	case "stdout":
		return stdout.New(o.FunctionThreshold, o.RequestThreshold), nil
	case "otel":
		return otel.New(logger, o.FunctionThreshold, o.RequestThreshold, o.OtelServiceName, o.OtelEndpoint)
	}

	return nil, fmt.Errorf("sink not found: %s", o.Sink)
}
