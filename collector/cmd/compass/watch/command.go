// Package watch for handling the watch command.
package watch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/skpr/compass/collector/cmd/compass/watch/app"
	"github.com/skpr/compass/collector/pkg/tracing"
)

const cmdLong = `
  Watch and analyze new profiles.`

const cmdExample = `
  # Watch and analyze new profiles.
  compass watch`

// Options is the commandline options for 'watch' sub command
type Options struct {
	Addr string
}

// NewOptions will return a new Options.
func NewOptions() Options {
	return Options{}
}

// NewCommand will return a new Cobra command.
func NewCommand() *cobra.Command {
	o := NewOptions()

	cmd := &cobra.Command{
		Use:                   "watch",
		DisableFlagsInUseLine: true,
		Short:                 "Watch and analyze new profiles.",
		Args:                  cobra.ExactArgs(0),
		Long:                  cmdLong,
		Example:               cmdExample,
		RunE: func(_ *cobra.Command, _ []string) error {
			return o.Run(o.Addr)
		},
	}

	cmd.Flags().StringVar(&o.Addr, "addr", ":27624", "Address to listen on for new profiles")

	return cmd
}

// Run will execute the dump command.
func (o *Options) Run(addr string) error {
	p := tea.NewProgram(app.NewModel(), tea.WithAltScreen())

	ctx, cancel := context.WithCancel(context.Background())

	eg := errgroup.Group{}

	router := http.NewServeMux()

	// Handle incoming profiling data.
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var profile tracing.Profile

		err = json.Unmarshal(body, &profile)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		p.Send(profile)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Profile received"))
	})

	server := &http.Server{Addr: addr, Handler: router}

	// Initialize the HTTP server which will receive profiling data.
	eg.Go(func() error {
		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("http server err: %w", err)
		}

		return nil
	})

	// Wait for the app/context to be cancelled and then shutdown the server.
	eg.Go(func() error {
		<-ctx.Done()
		return server.Shutdown(ctx)
	})

	// Start the application.
	eg.Go(func() error {
		_, err := p.Run()
		if err != nil {
			return fmt.Errorf("failed to run program: %w", err)
		}

		cancel()

		return nil
	})

	return eg.Wait()
}
