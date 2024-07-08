package manager

import (
	"fmt"
	"time"

	"github.com/patrickmn/go-cache"

	"github.com/skpr/compass/collector/internal/event/types"
)

type Client struct {
	// Consider an interface for the storage.
	storage *cache.Cache
}

func New(expire time.Duration) (*Client, error) {
	client := &Client{
		storage: cache.New(expire, expire),
	}

	return client, nil
}

func (c *Client) AddFunction(requestId, name string, executionTime uint64, expire time.Duration) error {
	function := types.Function{
		Name:          name,
		ExecutionTime: executionTime,
	}

	var functions []types.Function

	if x, found := c.storage.Get(requestId); found {
		functions = x.([]types.Function)
	}

	functions = append(functions, function)

	c.storage.Set(requestId, functions, expire)

	return nil
}

func (c *Client) FlushRequest(requestId string) (map[string]types.FunctionSummary, error) {
	defer c.storage.Delete(requestId)

	var functions []types.Function

	if x, found := c.storage.Get(requestId); found {
		functions = x.([]types.Function)
	}

	if len(functions) == 0 {
		return nil, fmt.Errorf("no functions found for request with id: %s", requestId)
	}

	summary := make(map[string]types.FunctionSummary)

	for _, function := range functions {
		f := types.FunctionSummary{
			TotalExecutionTime: function.ExecutionTime,
			Invocations:        1,
		}

		if _, ok := summary[function.Name]; ok {
			f.TotalExecutionTime = f.TotalExecutionTime + summary[function.Name].TotalExecutionTime
			f.Invocations = f.Invocations + summary[function.Name].Invocations
		}

		summary[function.Name] = f
	}

	return summary, nil
}
