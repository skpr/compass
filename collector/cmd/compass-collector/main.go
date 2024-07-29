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
		flagPlugin   string
		flagLibPath  string
		flagLogLevel string
	)

	cmd := &cobra.Command{
		Use:     "run",
		Short:   "Run the collector",
		Long:    cmdLong,
		Example: cmdExample,
		RunE: func(cmd *cobra.Command, args []string) error {
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

			return collector.Run(cmd.Context(), logger, flagLibPath, p)
		},
	}

	cmd.PersistentFlags().StringVar(&flagPlugin, "plugin", envget.GetString("COMPASS_COLLECTOR_PLUGIN", "/usr/lib64/compass/stdout.so"), "Plugin for processing tracing data")
	cmd.PersistentFlags().StringVar(&flagLibPath, "lib-path", "/usr/lib/php/modules/compass.so", "Path to the Compass extension")
	cmd.PersistentFlags().StringVar(&flagLogLevel, "debug", envget.GetString("COMPASS_COLLECTOR_LOG_LEVEL", "info"), "Set the logging level")

	err := cmd.Execute()
	if err != nil {
		panic(err)
	}
}
