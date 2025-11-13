// Package main provides the entrypoint for the sidecar.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/skpr/compass/collector"
	"github.com/skpr/compass/collector/extension/discovery"
	"github.com/skpr/compass/sidecar/broadcaster"
)

var (
	cmdLong = `
		A sidecar for dynamically observing applications.`

	cmdExample = `
		# Run the sidecar with the defaults.
		compass-sidecar

		# Enable debugging.
		export COMPASS_LOG_LEVEL=info
		compass-sidecar`
)

// Config utilised by this sidecar application.
type Config struct {
	Addr          string `yaml:"addr"           env:"COMPASS_SIDECAR_ADDR"           env-default:":8080"`
	LogLevel      string `yaml:"log_level"      env:"COMPASS_SIDECAR_LOG_LEVEL"      env-default:"info"`
	ProcessName   string `yaml:"log_level"      env:"COMPASS_SIDECAR_PROCESS_NAME"   env-default:"php-fpm"`
	ExtensionPath string `yaml:"extension_path" env:"COMPASS_SIDECAR_EXTENSION_PATH" env-default:"/usr/lib/php/modules/compass.so"`
}

// Options for this sidecar application.
type Options struct {
	// Path to the config file.
	Config string
}

func main() {
	o := Options{}

	cmd := &cobra.Command{
		Use:     "compass-sidecar",
		Short:   "Run the Compass sidecar",
		Long:    cmdLong,
		Example: cmdExample,
		RunE: func(cmd *cobra.Command, _ []string) error {
			var config Config

			err := cleanenv.ReadEnv(&config)
			if err != nil {
				return fmt.Errorf("failed to read config: %w", err)
			}

			lvl := new(slog.LevelVar)

			if err := lvl.UnmarshalText([]byte(config.LogLevel)); err != nil {
				lvl.Set(slog.LevelInfo)
			}

			logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
				Level: lvl,
			}))

			logger.Info("Looking for extension", "process_name", config.ProcessName)

			path, err := discovery.GetPathFromProcess(config.ProcessName, config.ExtensionPath)
			if err != nil {
				return err
			}

			logger.Info("Extension found", "process_name", config.ProcessName, "extension_path", path)

			b := broadcaster.New()

			eg, ctx := errgroup.WithContext(cmd.Context())

			var (
				collectorCtx    context.Context
				collectorCancel context.CancelFunc
			)

			// Loop for http server.
			eg.Go(func() error {
				mux := http.NewServeMux()

				mux.HandleFunc("/v1/traces", func(w http.ResponseWriter, r *http.Request) {
					subscriber := b.Subscribe()
					defer b.Unsubscribe(subscriber)

					w.Header().Set("Content-Type", "text/event-stream")
					w.Header().Set("Cache-Control", "no-cache")
					w.Header().Set("Connection", "keep-alive")
					w.WriteHeader(http.StatusOK)

					flusher, ok := w.(http.Flusher)
					if !ok {
						http.Error(w, "Streaming not supported", http.StatusInternalServerError)
						return
					}

					clientCtx := r.Context()

					for {
						select {
						case <-clientCtx.Done():
							fmt.Println("Client disconnected")
							return
						case msg, ok := <-subscriber:
							if !ok {
								fmt.Println("Subscriber channel closed")
								return
							}

							if err := json.NewEncoder(w).Encode(msg); err != nil {
								logger.Error("Failed to write to client", "error", err)
								return
							}

							flusher.Flush()
						}
					}
				})

				server := &http.Server{
					Addr:    config.Addr,
					Handler: mux,
				}

				// Start the server
				eg.Go(func() error {
					logger.Info("HTTP server listening", "addr", config.Addr)
					return server.ListenAndServe()
				})

				// Shutdown the server on context cancel
				<-ctx.Done()
				logger.Info("Shutting down HTTP server")

				shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				return server.Shutdown(shutdownCtx)
			})

			// Loop for starting the collector.
			eg.Go(func() error {
				for {
					if ctx == nil {
						return nil
					}

					if ctx.Done() == nil {
						return nil
					}

					// Booo! Needs to be better.
					time.Sleep(time.Second)

					if b.Subscribers() == 0 {
						continue
					}

					logger.Info("We have subscribers, starting collector")

					collectorCtx, collectorCancel = context.WithCancel(ctx)

					err := collector.Run(collectorCtx, logger, b, collector.RunOptions{
						ExecutablePath: path,
					})
					if err != nil {
						logger.Error("Failed to run collector", "error", err)
					}

					logger.Info("Collector has shutdown")
				}
			})

			// Loop for shutting down the collector.
			eg.Go(func() error {
				for {
					select {
					case <-ctx.Done():
						return nil // Sidecar is shutting down
					default:
						// Sleep to avoid spinning
						time.Sleep(time.Second)

						if collectorCancel == nil || collectorCtx == nil {
							continue
						}

						select {
						case <-collectorCtx.Done():
							// Already cancelled
							continue
						default:
							if b.Subscribers() > 0 {
								continue
							}

							logger.Info("No more subscribers, shutting down collector")

							collectorCancel()
							collectorCancel = nil
							collectorCtx = nil
						}
					}
				}
			})

			log.Println("Listening on:", config.Addr)

			return eg.Wait()
		},
	}

	// Command flags.
	cmd.PersistentFlags().StringVar(&o.Config, "config", "", "Path to the sidecar config file")

	err := cmd.Execute()
	if err != nil {
		panic(err)
	}
}
