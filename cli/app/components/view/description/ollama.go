package description

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ollama/ollama/api"
	"github.com/skpr/compass/trace"
)

const contextMessage = `Here is a PHP function call trace represented as JSON.

This JSON object has the following top level fields:

- metadata = contains all metadata about a trace
- functionCalls = all the function calls which occurred during a trace

The fields in metadata are:

- requestID = the request identifier associated with this trace
- uri = the request URI associated with this trace
- method = the HTTP method associated with this trace
- startTime = the start time associated with this trace
- endTime = the end time associated with this trace
- executionTime = the execution time associated with this trace

The fields in functionCalls are:

- name = the name of the PHP class and function being called
- startTime = the start time of the PHP function call
- endTime = the end time of the PHP function call

You are to analyse the following trace and determine:

- Where is all the request time being used?
- What are the most used function calls?
`

func getDescription(trace trace.Trace) (string, error) {
	// @todo, Should be passed in.
	client, err := api.ClientFromEnvironment()
	if err != nil {
		return "", err
	}

	jsonBytes, err := json.Marshal(trace)
	if err != nil {
		return "", err
	}

	messages := []api.Message{
		{
			Role:    "user",
			Content: contextMessage,
		},
		{
			Role:    "user",
			Content: fmt.Sprintf("Analyse this trace: %s", string(jsonBytes)),
		},
	}

	ctx := context.Background()

	req := &api.ChatRequest{
		Model:    "llama3.2",
		Messages: messages,
	}

	var content string

	err = client.Chat(ctx, req, func(resp api.ChatResponse) error {
		content = resp.Message.Content
		return nil
	})
	if err != nil {
		return "", err
	}

	return content, nil
}
