package manager

import (
	"fmt"
	"time"

	"github.com/skpr/compass/collector/internal/event/types"
)

type Client struct {
	requests types.Requests
}

func New() (*Client, error) {
	client := &Client{
		requests: make(types.Requests),
	}

	return client, nil
}

func (c *Client) AddFunction(requestId, name string, executionTime uint64, expire time.Duration) error {
	if _, found := c.requests[requestId]; !found {
		c.requests[requestId] = types.Request{
			ID:        requestId,
			Functions: make(types.Functions),
		}
	}

	if _, found := c.requests[requestId].Functions[name]; !found {
		c.requests[requestId].Functions[name] = types.Function{
			Name:          name,
			ExecutionTime: executionTime,
			Invocations:   1,
		}

		return nil
	}

	c.requests[requestId].Functions[name] = types.Function{
		Name:          name,
		ExecutionTime: c.requests[requestId].Functions[name].ExecutionTime + executionTime,
		Invocations:   c.requests[requestId].Functions[name].Invocations + 1,
	}

	return nil
}

// FlushRequest a request which has finished.
// @todo, Consider flushing out old unfinished requests here too (memory leak).
func (c *Client) FlushRequest(requestId string) (types.Request, error) {
	defer delete(c.requests, requestId)

	if _, found := c.requests[requestId]; !found {
		return types.Request{}, fmt.Errorf("request not found")
	}

	return c.requests[requestId], nil
}
