// Package stdout implements a simple sink that prints to stdout.
package stdout

import (
	"os"

	"encoding/json"

	"github.com/skpr/compass/trace"
)

// New client for handling traces to stdout.
func New(functionThreshold, requestThreshold int64) *Client {
	return &Client{
		functionThreshold: functionThreshold,
		requestThreshold:  requestThreshold,
	}
}

// Client for handling traces to stdout.
type Client struct {
	functionThreshold int64
	requestThreshold  int64
}

// Initialize the plugin.
func (c *Client) Initialize() error {
	return nil
}

// ProcessTrace from the collector.
func (c *Client) ProcessTrace(t trace.Trace) error {
	if t.ExecutionTime < c.requestThreshold {
		return nil
	}

	// Remove any duplicates.
	t = t.Dedupe()

	var calls []trace.FunctionCall

	for _, function := range t.FunctionCalls {
		executionTime := function.EndTime - function.StartTime

		if executionTime > c.functionThreshold {
			calls = append(calls, function)
		}
	}

	return json.NewEncoder(os.Stdout).Encode(t)
}