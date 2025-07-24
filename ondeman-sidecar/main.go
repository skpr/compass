package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

func main() {
	s := NewServer()

	if err := s.Run(); err != nil {
		panic(err)
	}
}

func (s *Server) Handle(w http.ResponseWriter, r *http.Request) {
	requestID := fmt.Sprintf("%d", time.Now().Unix())

	fmt.Println("Request in")

	s.Lock()
	s.handlers[requestID] = ""
	s.Unlock()

	time.Sleep(5 * time.Second)

	s.Lock()
	delete(s.handlers, requestID)
	s.Unlock()

	// Write a response back to the client
	fmt.Fprint(w, "OK\n")
}

type Server struct {
	sync.Mutex
	handlers map[string]string
}

func NewServer() *Server {
	return &Server{
		handlers: make(map[string]string),
	}
}

func (s *Server) Run() error {
	var eg errgroup.Group

	var (
		ctx    context.Context
		cancel context.CancelFunc
	)

	// Loop to check and run the collector.
	eg.Go(func() error {
		for {
			// We don't need to do anything if
			if len(s.handlers) == 0 {
				continue
			}

			ctx, cancel = context.WithCancel(context.Background())

			fmt.Println("Start the run process")

			runProcess(ctx)

			fmt.Println("Stop the run process")
		}
	})

	// A loop to stop the collector if we don't have any handlers.
	eg.Go(func() error {
		for {
			// We don't need to check anything if we have handlers.
			if len(s.handlers) > 0 {
				continue
			}

			if ctx == nil {
				continue
			}

			if ctx.Done() == nil {
				continue
			}

			// Check the current context. Cancel it if we don't have handlers.
			cancel()
		}
	})

	// Run the server.
	eg.Go(func() error {
		mux := http.NewServeMux()

		mux.HandleFunc("/", s.Handle)

		fmt.Println("Starting server")

		return http.ListenAndServe(":8080", mux)
	})

	return eg.Wait()
}

func runProcess(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			time.Sleep(time.Second)
		}
	}
}
