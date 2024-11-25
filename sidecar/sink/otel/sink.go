// Package otel implements a sink that sends trace data to OpenTelemetry.
package otel

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"net/http"

	"encoding/hex"
	"encoding/json"

	"github.com/skpr/compass/trace"
)

// New client for handling profiles to stdout.
func New(functionThreshold, requestThreshold int64, endpoint string) *Client {
	return &Client{
		functionThreshold: functionThreshold,
		requestThreshold:  requestThreshold,
		endpoint:          endpoint,
	}
}

// Client for handling profiles to stdout.
type Client struct {
	functionThreshold int64
	requestThreshold  int64
	endpoint          string
}

// Initialize the plugin.
func (c *Client) Initialize() error {
	return nil
}

// ProcessTrace from the collector.
func (c *Client) ProcessTrace(trace trace.Trace) error {
	if trace.ExecutionTime < c.requestThreshold {
		return nil
	}

	var spans []Span

	for _, function := range trace.Dedupe().FunctionCalls {
		spans = append(spans, Span{
			TraceID:           trace.RequestID,
			SpanID:            generateSpanID(),
			Name:              function.Name,
			Kind:              "SPAN_KIND_INTERNAL",
			StartTimeUnixNano: function.StartTime * 1000,
			EndTimeUnixNano:   function.EndTime * 1000,
		})
	}

	payloadBuf := new(bytes.Buffer)

	err := json.NewEncoder(payloadBuf).Encode(Trace{
		ResourceSpans: []ResourceSpan{
			{
				Resource: Resource{
					Attributes: []Attribute{
						{
							Key: "service.name",
							Value: AttributeValue{
								StringValue: "example",
							},
						},
						{
							Key: "uri",
							Value: AttributeValue{
								StringValue: trace.URI,
							},
						},
						{
							Key: "method",
							Value: AttributeValue{
								StringValue: trace.Method,
							},
						},
					},
				},
				ScopeSpans: []ScopeSpan{
					{
						Spans: spans,
					},
				},
			},
		},
	})
	if err != nil {
		return err
	}

	fmt.Println("sending to jaeger")

	req, err := http.NewRequest("POST", c.endpoint, payloadBuf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	res, e := client.Do(req)
	if e != nil {
		return e
	}

	defer res.Body.Close()

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	bodyString := string(bodyBytes)

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send trace data: %s", bodyString)
	}

	return nil
}

func generateSpanID() string {
	b := make([]byte, 8) // 8 bytes for spanId
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
