// Package main provides the entrypoint for the sidecar.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/skpr/compass/tracing/collector"
	"github.com/skpr/compass/tracing/collector/extension/discovery"
	"github.com/skpr/compass/tracing/sidecar/broadcaster"
)

var cmdExample = `
  # Run the sidecar with the defaults.
  compass-sidecar

  # Enable debugging.
  export COMPASS_LOG_LEVEL=info
  compass-sidecar`

var (
	metricCollectorRunning = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "compass_sidecar_collector_running",
		Help: "If the collector is running. 1 = on, 0 = off.",
	})

	metricSubscription = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "compass_sidecar_subscriptions",
		Help: "The total number of currently subscribed streams",
	})
)

// Config utilised by this sidecar application.
type Config struct {
	Addr          string `yaml:"addr"           env:"COMPASS_SIDECAR_ADDR"           env-default:":28624"`
	LogLevel      string `yaml:"log_level"      env:"COMPASS_SIDECAR_LOG_LEVEL"      env-default:"info"`
	ProcessName   string `yaml:"log_level"      env:"COMPASS_SIDECAR_PROCESS_NAME"   env-default:"php-fpm"`
	ExtensionPath string `yaml:"extension_path" env:"COMPASS_SIDECAR_EXTENSION_PATH" env-default:"/usr/lib/php/modules/compass.so"`
	Token         string `yaml:"token"          env:"COMPASS_SIDECAR_TOKEN"`
	CertFile      string `yaml:"cert_file"      env:"COMPASS_SIDECAR_CERT_FILE"`
	KeyFile       string `yaml:"key_file"       env:"COMPASS_SIDECAR_KEY_FILE"`
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
		Long:    "A sidecar for dynamically observing applications.",
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

				mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
					promhttp.Handler().ServeHTTP(w, r)
				})

				mux.HandleFunc("/v1/traces", func(w http.ResponseWriter, r *http.Request) {
					if config.Token != "" && config.Token != r.Header.Get("X-Skpr-Token") {
						w.WriteHeader(http.StatusUnauthorized)
						fmt.Fprintln(w, "Access Denied")
						return
					}

					// Track the number of subscriptions for debugging how many clients are using the sidecar.
					metricSubscription.Inc()
					defer metricSubscription.Dec()

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
								// Treat client-context cancellation as a normal disconnect.
								if errors.Is(clientCtx.Err(), context.Canceled) {
									logger.Info("Client write failed due to context cancellation")
									return
								}

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

				// Start the server in its own goroutine.
				eg.Go(func() error {

					listenAndServe := func(certFile, keyFile string) error {
						if config.CertFile != "" && config.KeyFile != "" {
							logger.Info("Server listening with TLS", "addr", config.Addr)

							return server.ListenAndServeTLS(config.CertFile, config.KeyFile)
						}

						logger.Info("Server listening", "addr", config.Addr)

						return server.ListenAndServe()
					}

					if err := listenAndServe(config.CertFile, config.KeyFile); err != nil && !errors.Is(err, http.ErrServerClosed) {
						return err
					}

					return nil
				})

				// Shutdown the server on context cancel.
				<-ctx.Done()
				logger.Info("Shutting down HTTP server")

				shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				if err := server.Shutdown(shutdownCtx); err != nil &&
					!errors.Is(err, context.Canceled) &&
					!errors.Is(err, context.DeadlineExceeded) {
					return err
				}

				return nil
			})

			// Loop for starting the collector.
			eg.Go(func() error {
				for {
					select {
					case <-ctx.Done():
						logger.Info("Collector loop exiting due to context cancellation")
						return nil
					default:
					}

					// Avoid spinning.
					time.Sleep(time.Second)

					if b.Subscribers() == 0 {
						continue
					}

					logger.Info("We have subscribers, starting collector")

					// Track when our collector is running for debugging.
					metricCollectorRunning.Set(1)

					collectorCtx, collectorCancel = context.WithCancel(ctx)

					err := collector.Run(collectorCtx, logger, b, collector.RunOptions{
						ExecutablePath: path,
					})
					if err != nil && !errors.Is(err, context.Canceled) {
						logger.Error("Failed to run collector", "error", err)
					}

					metricCollectorRunning.Set(0)

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

							logger.Info("No more subscribers, triggering collector shutdown")

							collectorCancel()
							collectorCancel = nil
							collectorCtx = nil
						}
					}
				}
			})

			if err := eg.Wait(); err != nil && !errors.Is(err, context.Canceled) {
				return err
			}

			return nil
		},
	}

	// Command flags.
	cmd.PersistentFlags().StringVar(&o.Config, "config", "", "Path to the sidecar config file")

	err := cmd.Execute()
	if err != nil {
		panic(err)
	}
}
