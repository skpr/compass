// Package main provides the entrypoint for the sidecar.
package main

import (
	"log/slog"
	"os"
	"time"

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

		# Run the sidecar and target a different library.
		compass-sidecar --extension=/usr/lib/php/modules/something-else.so`
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
	cmd.PersistentFlags().StringVar(&o.ProcessName, "process-name", "php-fpm", "Name of the process which will be used for discovery")
	cmd.PersistentFlags().DurationVar(&o.ProcessPoll, "process-poll", time.Second*5, "How frequently to poll for current list of processes")
	cmd.PersistentFlags().StringVar(&o.ExtensionPath, "extension-path", "/usr/lib/php/modules/compass.so", "Path to the Compass extension")

	// Sink configuration.
	// @todo, Configurable.
	cmd.PersistentFlags().Int64Var(&o.FunctionThreshold, "function-threshold", 10, "Path to the Compass extension")
	cmd.PersistentFlags().Int64Var(&o.RequestThreshold, "request-threshold", 100, "Path to the Compass extension")

	// Debugging.
	cmd.PersistentFlags().StringVar(&o.LogLevel, "log-level", "info", "Set the logging level")

	err := cmd.Execute()
	if err != nil {
		panic(err)
	}
}