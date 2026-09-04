// Package main provides the entrypoint for the sidecar.
package main

import (
	"context"
	"crypto/subtle"
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

	nodediscovery "github.com/skpr/compass/pkg/node/addon/discovery"
	phpdiscovery "github.com/skpr/compass/pkg/php/extension/discovery"
	"github.com/skpr/compass/pkg/tracer"
)

var cmdExample = `
  # Run the sidecar with the defaults.
  compass-sidecar

  # Run the sidecar with a config file.
  compass-sidecar --config=/etc/compass/sidecar.yaml

  # Enable debugging.
  export COMPASS_SIDECAR_LOG_LEVEL=debug
  compass-sidecar`

// HeaderToken is the header this sidecar authenticates requests with.
const HeaderToken = "X-Skpr-Token"

// authorized reports whether a request may proceed. An empty configured token
// disables authentication; otherwise the presented token must match in
// constant time so a rejection does not leak how many leading bytes were right.
func authorized(want, got string) bool {
	if want == "" {
		return true
	}

	return subtle.ConstantTimeCompare([]byte(want), []byte(got)) == 1
}

var (
	serverReadHeaderTimeout = 10 * time.Second
	serverIdleTimeout       = 2 * time.Minute
)

func newServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:    addr,
		Handler: handler,
		// No WriteTimeout: /v1/traces streams for the life of the subscription,
		// so slow clients are bounded by these two rather than a write deadline.
		ReadHeaderTimeout: serverReadHeaderTimeout,
		IdleTimeout:       serverIdleTimeout,
	}
}

var (
	metricCollectorRunning = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "compass_sidecar_collector_running",
		Help: "If the collector is running. 1 = on, 0 = off.",
	})

	metricSubscription = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "compass_sidecar_subscriptions",
		Help: "The total number of currently subscribed streams",
	})

	metricRuntimeDiscovered = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "compass_sidecar_runtime_discovered",
		Help: "If a runtime was discovered and is being traced. 1 = yes, 0 = no.",
	}, []string{"runtime"})

	metricTracesDropped = promauto.NewCounter(prometheus.CounterOpts{
		Name: "compass_sidecar_traces_dropped_total",
		Help: "The total number of traces dropped because a subscriber could not keep up.",
	})
)

// Config utilised by this sidecar application.
type Config struct {
	Addr             string        `yaml:"addr"               env:"COMPASS_SIDECAR_ADDR"               env-default:":28624"`
	LogLevel         string        `yaml:"log_level"          env:"COMPASS_SIDECAR_LOG_LEVEL"          env-default:"info"`
	PHPProcessName   string        `yaml:"php_process_name"   env:"COMPASS_SIDECAR_PHP_PROCESS_NAME"   env-default:"php-fpm"`
	PHPExtensionPath string        `yaml:"php_extension_path" env:"COMPASS_SIDECAR_PHP_EXTENSION_PATH" env-default:"/usr/lib/php/modules/compass.so"`
	NodeProcessName  string        `yaml:"node_process_name"  env:"COMPASS_SIDECAR_NODE_PROCESS_NAME"  env-default:"node"`
	NodeAddonPath    string        `yaml:"node_addon_path"    env:"COMPASS_SIDECAR_NODE_ADDON_PATH"    env-default:"/usr/lib/compass/node/compass.node"`
	DiscoveryTimeout time.Duration `yaml:"discovery_timeout"  env:"COMPASS_SIDECAR_DISCOVERY_TIMEOUT"  env-default:"1m"`
	MaxFunctionCalls int           `yaml:"max_function_calls" env:"COMPASS_SIDECAR_MAX_FUNCTION_CALLS" env-default:"10000"`
	Token            string        `yaml:"token"              env:"COMPASS_SIDECAR_TOKEN"`
	CertFile         string        `yaml:"cert_file"          env:"COMPASS_SIDECAR_CERT_FILE"`
	KeyFile          string        `yaml:"key_file"           env:"COMPASS_SIDECAR_KEY_FILE"`
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
		// Usage is not helpful for a runtime failure.
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			config, err := loadConfig(o.Config)
			if err != nil {
				return err
			}

			lvl := new(slog.LevelVar)

			if err := lvl.UnmarshalText([]byte(config.LogLevel)); err != nil {
				lvl.Set(slog.LevelInfo)
			}

			logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
				Level: lvl,
			}))
			slog.SetDefault(logger)

			runtimes, err := discoverRuntimes(cmd.Context(), logger, config)
			if err != nil {
				return err
			}

			b := NewBroadcaster()

			eg, ctx := errgroup.WithContext(cmd.Context())

			// Loop for http server.
			eg.Go(func() error {
				mux := http.NewServeMux()

				mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
					promhttp.Handler().ServeHTTP(w, r)
				})

				mux.HandleFunc("/v1/traces", func(w http.ResponseWriter, r *http.Request) {
					if !authorized(config.Token, r.Header.Get(HeaderToken)) {
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
							logger.Info("Client disconnected")
							return
						case msg, ok := <-subscriber:
							if !ok {
								logger.Info("Subscriber channel closed")
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

				server := newServer(config.Addr, mux)

				// Start the server in its own goroutine.
				eg.Go(func() error {
					listenAndServe := func(certFile, keyFile string) error {
						if certFile != "" && keyFile != "" {
							logger.Info("Server listening with TLS", "addr", config.Addr)

							return server.ListenAndServeTLS(certFile, keyFile)
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

			supervisor := newCollectorSupervisor(logger, b, func(collectorCtx context.Context) error {
				return tracer.Run(collectorCtx, b, runtimes, tracer.Options{MaxFunctionCalls: config.MaxFunctionCalls})
			})

			eg.Go(func() error {
				return supervisor.Run(ctx)
			})

			if err := eg.Wait(); err != nil && !errors.Is(err, context.Canceled) {
				return err
			}

			return nil
		},
	}

	// Command flags.
	cmd.PersistentFlags().StringVar(&o.Config, "config", "", "Path to the sidecar config file")

	// Cobra prints the error, so exit quietly rather than panicking with a
	// stack trace over the top of it.
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// loadConfig from a file, if one was provided, with the environment taking precedence.
func loadConfig(path string) (Config, error) {
	var config Config

	if path != "" {
		if err := cleanenv.ReadConfig(path, &config); err != nil {
			return config, fmt.Errorf("failed to read config file %s: %w", path, err)
		}

		return config, nil
	}

	if err := cleanenv.ReadEnv(&config); err != nil {
		return config, fmt.Errorf("failed to read config: %w", err)
	}

	return config, nil
}

// discoverRuntimes which are present and instrumented.
//
// Each runtime is optional, a deployment usually only runs one of them, so a
// runtime which is not present is skipped instead of failing the sidecar.
func discoverRuntimes(ctx context.Context, logger *slog.Logger, config Config) (tracer.Runtimes, error) {
	var (
		phpExtensionPath string
		nodeAddonPath    string
		eg               errgroup.Group
	)

	metricRuntimeDiscovered.WithLabelValues("php").Set(0)
	metricRuntimeDiscovered.WithLabelValues("node").Set(0)

	eg.Go(func() error {
		logger.Info("Looking for PHP extension", "php_process_name", config.PHPProcessName)

		path, err := phpdiscovery.GetPathFromProcess(ctx, config.PHPProcessName, config.PHPExtensionPath, config.DiscoveryTimeout)
		if err != nil {
			if errors.Is(err, phpdiscovery.ErrNotFound) {
				logger.Info("PHP extension not found, skipping PHP", "php_process_name", config.PHPProcessName, "error", err)
				return nil
			}

			return err
		}

		logger.Info("PHP extension found", "php_process_name", config.PHPProcessName, "php_extension_path", path)

		phpExtensionPath = path

		metricRuntimeDiscovered.WithLabelValues("php").Set(1)

		return nil
	})

	eg.Go(func() error {
		logger.Info("Looking for Node addon", "node_process_name", config.NodeProcessName)

		path, err := nodediscovery.GetPathFromProcess(ctx, config.NodeProcessName, config.NodeAddonPath, config.DiscoveryTimeout)
		if err != nil {
			if errors.Is(err, nodediscovery.ErrNotFound) {
				logger.Info("Node addon not found, skipping Node", "node_process_name", config.NodeProcessName, "error", err)
				return nil
			}

			return err
		}

		logger.Info("Node addon found", "node_process_name", config.NodeProcessName, "node_addon_path", path)

		nodeAddonPath = path

		metricRuntimeDiscovered.WithLabelValues("node").Set(1)

		return nil
	})

	if err := eg.Wait(); err != nil {
		return tracer.Runtimes{}, err
	}

	runtimes := tracer.Runtimes{
		PHPExtensionPath: phpExtensionPath,
		NodeAddonPath:    nodeAddonPath,
	}

	if runtimes.Empty() {
		return runtimes, fmt.Errorf("no instrumented runtimes found: looked for the PHP extension at %s (process %q) and the Node addon at %s (process %q)",
			config.PHPExtensionPath, config.PHPProcessName, config.NodeAddonPath, config.NodeProcessName)
	}

	return runtimes, nil
}
