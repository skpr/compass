// Package ollama provides an HTTP client for the Ollama API.
package ollama

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/skpr/compass/pkg/trace"
)

const systemPrompt = `You are a performance analyst reviewing application traces. Analyze the following trace data and identify:

1. Bottlenecks - functions consuming disproportionate execution time
2. Performance issues - repeated calls, slow operations, or inefficient patterns
3. Optimization opportunities - actionable suggestions to improve performance

Be concise and actionable. Focus on the most impactful findings. Format your response with clear sections using markdown headers.`

// Client for communicating with the Ollama API.
type Client struct {
	url   string
	model string
}

// NewClient creates a new Ollama API client.
func NewClient(url, model string) *Client {
	return &Client{
		url:   strings.TrimRight(url, "/"),
		model: model,
	}
}

// generateRequest is the request body for the Ollama generate API.
type generateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	System string `json:"system"`
	Stream bool   `json:"stream"`
}

// generateResponse is a single response chunk from the Ollama generate API.
type generateResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
	Error    string `json:"error,omitempty"`
}

// Summarize sends a trace to Ollama and returns a performance analysis summary.
func (c *Client) Summarize(t trace.Trace) (string, error) {
	traceJSON, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal trace: %w", err)
	}

	prompt := fmt.Sprintf("Analyze this application trace:\n\n```json\n%s\n```", string(traceJSON))

	reqBody := generateRequest{
		Model:  c.model,
		Prompt: prompt,
		System: systemPrompt,
		Stream: true,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest(http.MethodPost, c.url+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 5 * time.Minute,
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to send request to Ollama at %s: %w", c.url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama returned status %d", resp.StatusCode)
	}

	var result strings.Builder

	scanner := bufio.NewScanner(resp.Body)

	for scanner.Scan() {
		var chunk generateResponse

		if err := json.Unmarshal(scanner.Bytes(), &chunk); err != nil {
			return "", fmt.Errorf("failed to parse response chunk: %w", err)
		}

		if chunk.Error != "" {
			return "", fmt.Errorf("ollama error: %s", chunk.Error)
		}

		result.WriteString(chunk.Response)

		if chunk.Done {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	return result.String(), nil
}
