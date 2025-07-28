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

	"github.com/christgf/env"
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

// Options for this sidecar application.
type Options struct {
	Addr          string
	ProcessName   string
	ExtensionPath string
	LogLevel      string
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

			path, err := discovery.GetPathFromProcess(o.ProcessName, o.ExtensionPath)
			if err != nil {
				return err
			}

			logger.Info("Extension found", "process_name", o.ProcessName, "extension_path", path)

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
					Addr:    o.Addr,
					Handler: mux,
				}

				// Start the server
				eg.Go(func() error {
					logger.Info("HTTP server listening", "addr", o.Addr)
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

			log.Println("Listening on:", o.Addr)

			return eg.Wait()
		},
	}

	// Command flags.
	cmd.PersistentFlags().StringVar(&o.Addr, "addr", env.String("COMPASS_SIDECAR_ADDR", ":28624"), "Address to listen on for incoming requests")
	cmd.PersistentFlags().StringVar(&o.LogLevel, "log-level", env.String("COMPASS_SIDECAR_LOG_LEVEL", "info"), "Set the logging level")

	// Extension discovery flags.
	cmd.PersistentFlags().StringVar(&o.ProcessName, "process-name", env.String("COMPASS_PROCESS_NAME", "php-fpm"), "Name of the process which will be used for discovery")
	cmd.PersistentFlags().StringVar(&o.ExtensionPath, "extension-path", env.String("COMPASS_EXTENSION_PATH", "/usr/lib/php/modules/compass.so"), "Path to the Compass extension")

	err := cmd.Execute()
	if err != nil {
		panic(err)
	}
}
