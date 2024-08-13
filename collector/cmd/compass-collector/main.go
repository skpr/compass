// Package main provides the entrypoint for the collector.
package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/skpr/compass/collector/internal/collector"
	"github.com/skpr/compass/collector/internal/envget"
	"github.com/skpr/compass/collector/plugin"
)

var (
	cmdLong = `
		Run the process which collects tracing data.`

	cmdExample = `
		# Run the collector and target a specific container.
		compass-collector --container=php-fpm

		# Run the collector and target a different library.
		compass-collector --lib-path=/usr/lib/php/modules/something-else.so`
)

func main() {
	var (
		flagPlugin            string
		flagLibPath           string
		flagLogLevel          string
		flagRequestThreshold  float64
		flagFunctionThreshold float64
	)

	cmd := &cobra.Command{
		Use:     "run",
		Short:   "Run the collector",
		Long:    cmdLong,
		Example: cmdExample,
		RunE: func(cmd *cobra.Command, _ []string) error {
			lvl := new(slog.LevelVar)

			switch flagLogLevel {
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

			p, err := plugin.Load(flagPlugin)
			if err != nil {
				return fmt.Errorf("failed to load plugin: %w", err)
			}

			return collector.Run(cmd.Context(), logger, p, collector.RunOptions{
				ExecutablePath:    flagLibPath,
				RequestThreshold:  flagRequestThreshold,
				FunctionThreshold: flagFunctionThreshold,
			})
		},
	}

	cmd.PersistentFlags().StringVar(&flagPlugin, "plugin", envget.String("COMPASS_COLLECTOR_PLUGIN", "/usr/lib64/compass/stdout.so"), "Plugin for processing tracing data")
	cmd.PersistentFlags().StringVar(&flagLibPath, "lib-path", "/usr/lib/php/modules/compass.so", "Path to the Compass extension")
	cmd.PersistentFlags().StringVar(&flagLogLevel, "debug", envget.String("COMPASS_COLLECTOR_LOG_LEVEL", "info"), "Set the logging level")
	cmd.PersistentFlags().Float64Var(&flagRequestThreshold, "request-threshold", 100, "Process requests over this threshold")
	cmd.PersistentFlags().Float64Var(&flagFunctionThreshold, "function-threshold", 10, "Process summarised functions over this threshold")

	err := cmd.Execute()
	if err != nil {
		panic(err)
	}
}
