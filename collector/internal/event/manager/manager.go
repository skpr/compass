package manager

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/skpr/compass/collector/internal/event/types"
)

type Client struct {
	storage map[string]StorageItem
	mu      sync.RWMutex
}

type StorageItem struct {
	Expiration int64
	Functions  []types.Function
}

func New() (*Client, error) {
	client := &Client{
		storage: make(map[string]StorageItem),
	}

	return client, nil
}

func (c *Client) RunWithExpiration(ctx context.Context, interval time.Duration) error {
	ticker := time.NewTicker(interval)

	for {
		select {
		case <-ctx.Done():
			ticker.Stop()
			return nil
		case <-ticker.C:
			err := c.DeleteExpired()
			if err != nil {
				return err
			}
		}
	}
}

func (c *Client) AddFunction(requestId, name string, executionTime uint64, expire time.Duration) error {
	function := types.Function{
		Name:          name,
		ExecutionTime: executionTime,
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if val, ok := c.storage[requestId]; ok {
		val.Functions = append(val.Functions, function)
		c.storage[requestId] = val
		return nil
	}

	c.storage[requestId] = StorageItem{
		Expiration: time.Now().Add(expire).UnixNano(),
		Functions: []types.Function{
			function,
		},
	}

	return nil
}

func (c *Client) FlushRequest(requestId string) ([]types.Function, error) {
	if _, ok := c.storage[requestId]; !ok {
		return nil, fmt.Errorf("cannot find functions assocaited with request with id: %s", requestId)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	request := c.storage[requestId]

	// Cleanup after ourselves.
	delete(c.storage, requestId)

	return request.Functions, nil
}

func (c *Client) DeleteExpired() error {
	now := time.Now().UnixNano()

	c.mu.Lock()
	defer c.mu.Unlock()

	for k, v := range c.storage {
		if v.Expiration > 0 && now > v.Expiration {
			delete(c.storage, k)
		}
	}

	return nil
}
