// Package main provides the entrypoint for the DaemonSet collector.
//
// Unlike the sidecar, which shares one pod's PID namespace and traces that pod,
// the daemon runs once per node in the host PID namespace and traces a single
// pod chosen per connection. A client selects the pod by its UID on the
// request: GET /v1/traces?uid=<podUID>. The daemon resolves that UID to the
// pod's processes on the node by scanning /proc, attaches the eBPF probes to
// the runtime binary in the target container, and filters events to the pod's
// cgroups so a binary whose inode is shared across pods only reports the one
// asked for.
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
	"os/signal"
	"syscall"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/skpr/compass/pkg/collector"
	"github.com/skpr/compass/pkg/collector/pod"
	"github.com/skpr/compass/pkg/tracer"
	"github.com/skpr/compass/pkg/tracer/cgroupfilter"
	"github.com/skpr/compass/pkg/tracer/sink"
)

var cmdExample = `
  # Run the daemon (a token is mandatory).
  export COMPASS_DAEMON_TOKEN=$(cat /var/run/secrets/compass/token)
  compass-daemon

  # Enable debugging.
  export COMPASS_DAEMON_LOG_LEVEL=debug
  compass-daemon`

// HeaderToken is the header this daemon authenticates requests with.
const HeaderToken = "X-Skpr-Token"

// authorized reports whether a request may proceed. The token is mandatory in
// daemon mode (see Config.validate), so want is never empty here; the compare
// is constant time so a rejection does not leak how many leading bytes matched.
func authorized(want, got string) bool {
	return subtle.ConstantTimeCompare([]byte(want), []byte(got)) == 1
}

func requireToken(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !authorized(token, r.Header.Get(HeaderToken)) {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprintln(w, "Access Denied")
			return
		}

		next.ServeHTTP(w, r)
	})
}

var (
	serverReadHeaderTimeout = 10 * time.Second
	serverIdleTimeout       = 2 * time.Minute
)

func newServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:    addr,
		Handler: handler,
		// No WriteTimeout: /v1/traces streams for the life of the subscription.
		ReadHeaderTimeout: serverReadHeaderTimeout,
		IdleTimeout:       serverIdleTimeout,
	}
}

var (
	metricSubscription = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "compass_daemon_subscriptions",
		Help: "The number of currently subscribed streams.",
	})

	metricCollectorsRunning = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "compass_daemon_collectors_running",
		Help: "The number of per-pod collectors currently running.",
	})

	metricTracesDropped = promauto.NewCounter(prometheus.CounterOpts{
		Name: "compass_daemon_traces_dropped_total",
		Help: "The total number of traces dropped because a subscriber could not keep up.",
	})
)

// Config utilised by this daemon application.
type Config struct {
	Addr             string `yaml:"addr"               env:"COMPASS_DAEMON_ADDR"               env-default:":28624"`
	LogLevel         string `yaml:"log_level"          env:"COMPASS_DAEMON_LOG_LEVEL"          env-default:"info"`
	PHPExtensionPath string `yaml:"php_extension_path" env:"COMPASS_DAEMON_PHP_EXTENSION_PATH" env-default:"/usr/lib/php/modules/compass.so"`
	NodeAddonPath    string `yaml:"node_addon_path"    env:"COMPASS_DAEMON_NODE_ADDON_PATH"    env-default:"/usr/lib/compass/node/compass.node"`
	ProcRoot         string `yaml:"proc_root"          env:"COMPASS_DAEMON_PROC_ROOT"          env-default:"/proc"`
	CgroupRoot       string `yaml:"cgroup_root"        env:"COMPASS_DAEMON_CGROUP_ROOT"        env-default:"/sys/fs/cgroup"`
	MaxFunctionCalls int    `yaml:"max_function_calls" env:"COMPASS_DAEMON_MAX_FUNCTION_CALLS" env-default:"10000"`
	Token            string `yaml:"token"              env:"COMPASS_DAEMON_TOKEN"`
	CertFile         string `yaml:"cert_file"          env:"COMPASS_DAEMON_CERT_FILE"`
	KeyFile          string `yaml:"key_file"           env:"COMPASS_DAEMON_KEY_FILE"`
}

// validate rejects a configuration the daemon cannot safely run with.
//
// The token is mandatory: the daemon can trace any pod on the node by UID, so
// an unauthenticated daemon would let any client on the network read any pod's
// request data. That is a far larger blast radius than the per-pod sidecar, so
// there is no "empty token disables auth" escape hatch here.
func (c Config) validate() error {
	if c.Token == "" {
		return errors.New("COMPASS_DAEMON_TOKEN is required: the daemon can trace any pod on the node, so it will not serve without a token")
	}

	return nil
}

// Options for this daemon application.
type Options struct {
	// Path to the config file.
	Config string
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	o := Options{}

	cmd := &cobra.Command{
		Use:          "compass-daemon",
		Short:        "Run the Compass node daemon",
		Long:         "A per-node collector that traces a single pod, selected by UID, on demand.",
		Example:      cmdExample,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			config, err := loadConfig(o.Config)
			if err != nil {
				return err
			}

			if err := config.validate(); err != nil {
				return err
			}

			lvl := new(slog.LevelVar)
			if err := lvl.UnmarshalText([]byte(config.LogLevel)); err != nil {
				lvl.Set(slog.LevelInfo)
			}

			logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: lvl}))
			slog.SetDefault(logger)

			eg, ctx := errgroup.WithContext(cmd.Context())

			router := collector.NewRouter(ctx, logger, podCollector(logger, config),
				collector.WithMetrics(
					func(collector.Target) { metricCollectorsRunning.Inc() },
					func(collector.Target) { metricCollectorsRunning.Dec() },
					func(collector.Target) { metricTracesDropped.Inc() },
				),
			)

			eg.Go(func() error {
				return serve(ctx, logger, config, router)
			})

			if err := eg.Wait(); err != nil && !errors.Is(err, context.Canceled) {
				return err
			}

			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&o.Config, "config", "", "Path to the daemon config file")

	if err := cmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}

// podCollector builds the per-pod collector the router runs for each target: it
// resolves the pod's processes on the node, locates the instrumented runtime in
// the target container, and traces it with the events filtered to the pod's
// cgroups.
func podCollector(logger *slog.Logger, config Config) collector.CollectorFunc {
	resolver := &pod.Resolver{ProcRoot: config.ProcRoot, CgroupRoot: config.CgroupRoot}

	return func(ctx context.Context, target collector.Target, s sink.Interface) error {
		matches, err := resolver.Resolve(target.UID)
		if err != nil {
			return fmt.Errorf("failed to resolve pod %s: %w", target.UID, err)
		}

		if len(matches) == 0 {
			// The pod is not on this node, or not yet. Returning an error lets the
			// router retry with backoff, so a pod which starts shortly after the
			// client connects is picked up without the client reconnecting.
			return fmt.Errorf("pod %s has no processes on this node", target.UID)
		}

		runtimes := collector.LocateRuntimes(matches, config.PHPExtensionPath, config.NodeAddonPath)
		if runtimes.Empty() {
			return fmt.Errorf("pod %s has no instrumented runtime (looked for the PHP extension at %s and the Node addon at %s)",
				target.UID, config.PHPExtensionPath, config.NodeAddonPath)
		}

		logger.Info("Tracing pod",
			"uid", target.UID,
			"pids", len(matches),
			"php", runtimes.PHPExtensionPath != "",
			"node", runtimes.NodeAddonPath != "",
			"cgroups", len(runtimes.AllowedCgroupIDs),
		)

		return tracer.Run(ctx, s, tracer.Runtimes{
			PHPExtensionPath: runtimes.PHPExtensionPath,
			NodeAddonPath:    runtimes.NodeAddonPath,
		}, tracer.Options{
			MaxFunctionCalls: config.MaxFunctionCalls,
			Filter:           cgroupfilter.Filter{AllowedCgroupIDs: runtimes.AllowedCgroupIDs},
		})
	}
}

// serve runs the HTTP server until ctx is cancelled.
func serve(ctx context.Context, logger *slog.Logger, config Config, router *collector.Router) error {
	mux := http.NewServeMux()

	mux.Handle("/metrics", requireToken(config.Token, promhttp.Handler()))
	mux.Handle("/v1/traces", requireToken(config.Token, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleTraces(logger, router, w, r)
	})))

	server := newServer(config.Addr, mux)

	go func() {
		<-ctx.Done()
		logger.Info("Shutting down HTTP server")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil &&
			!errors.Is(err, context.Canceled) &&
			!errors.Is(err, context.DeadlineExceeded) {
			logger.Error("Failed to shut down HTTP server", "error", err)
		}
	}()

	listenAndServe := func() error {
		if config.CertFile != "" && config.KeyFile != "" {
			logger.Info("Server listening with TLS", "addr", config.Addr)
			return server.ListenAndServeTLS(config.CertFile, config.KeyFile)
		}

		logger.Info("Server listening", "addr", config.Addr)
		return server.ListenAndServe()
	}

	if err := listenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

// handleTraces streams the target pod's traces to a single client.
func handleTraces(logger *slog.Logger, router *collector.Router, w http.ResponseWriter, r *http.Request) {
	target, err := collector.ParseTarget(r.URL.Query())
	if err != nil {
		// A missing or malformed UID is a client error: the daemon traces a named
		// pod, not the whole node.
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	metricSubscription.Inc()
	defer metricSubscription.Dec()

	traces, unsubscribe := router.Subscribe(target)
	defer unsubscribe()

	w.Header().Set("Content-Type", "application/x-ndjson")
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
			logger.Info("Client disconnected", "uid", target.UID)
			return
		case msg, ok := <-traces:
			if !ok {
				logger.Info("Subscriber channel closed", "uid", target.UID)
				return
			}

			if err := json.NewEncoder(w).Encode(msg); err != nil {
				if errors.Is(clientCtx.Err(), context.Canceled) {
					return
				}

				logger.Error("Failed to write to client", "uid", target.UID, "error", err)
				return
			}

			flusher.Flush()
		}
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
