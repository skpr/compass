// Package main provides the entrypoint for the sidecar.
package main

import (
	"log/slog"
	"os"
	"time"

	"github.com/christgf/env"
	"github.com/spf13/cobra"

	"github.com/skpr/compass/collector"
	"github.com/skpr/compass/collector/extension/discovery"
	"github.com/skpr/compass/sidecar/sink"
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
	ProcessPoll       time.Duration
	ExtensionPath     string
	LogLevel          string
	FunctionThreshold int64
	RequestThreshold  int64
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

			path, err := discovery.GetPathFromProcess(logger, o.ProcessName, o.ExtensionPath, o.ProcessPoll)
			if err != nil {
				return err
			}

			logger.Info("Extension found", "process_name", o.ProcessName, "extension_path", path)

			logger.Info("Starting collector")

			err = collector.Run(cmd.Context(), logger, sink.New(o.FunctionThreshold, o.RequestThreshold), collector.RunOptions{
				ExecutablePath: path,
			})

			logger.Info("Collector finished")

			return nil
		},
	}

	// Extension discovery flags.
	cmd.PersistentFlags().StringVar(&o.ProcessName, "process-name", env.String("COMPASS_PROCESS_NAME", "php-fpm"), "Name of the process which will be used for discovery")
	cmd.PersistentFlags().DurationVar(&o.ProcessPoll, "process-poll", env.Duration("COMPASS_PROCESS_POLL", time.Second*5), "How frequently to poll for current list of processes")
	cmd.PersistentFlags().StringVar(&o.ExtensionPath, "extension-path", env.String("COMPASS_EXTENSION_PATH", "/usr/lib/php/modules/compass.so"), "Path to the Compass extension")

	// Sink configuration.
	cmd.PersistentFlags().Int64Var(&o.FunctionThreshold, "function-threshold", env.Int64("COMPASS_FUNCTION_THRESHOLD", 10), "Watermark for which functionss to trace")
	cmd.PersistentFlags().Int64Var(&o.RequestThreshold, "request-threshold", env.Int64("COMPASS_REQUEST_THRESHOLD", 100), "Watermark for which requests to trace")

	// Debugging.
	cmd.PersistentFlags().StringVar(&o.LogLevel, "log-level", env.String("COMPASS_LOG_LEVEL", "info"), "Set the logging level")

	err := cmd.Execute()
	if err != nil {
		panic(err)
	}
}
