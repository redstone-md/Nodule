// Package llmmock provides a test double for the llm.Client interface.
// It captures requests and returns configurable responses.
package llmmock

import (
	"context"

	"github.com/redstone-md/nodule/internal/llm"
)

// Client is a test double for llm.Client.
// It captures the last request and returns a preconfigured response.
type Client struct {
	name      string
	resp      *llm.GenerateResponse
	err       error
	lastReq   llm.GenerateRequest
	callCount int
}

// New creates a mock client that returns the given response on Generate.
func New(name string, resp *llm.GenerateResponse) *Client {
	return &Client{name: name, resp: resp}
}

// NewWithErr creates a mock client that returns an error on Generate.
func NewWithErr(name string, err error) *Client {
	return &Client{name: name, err: err}
}

// Generate records the request and returns the configured response or error.
func (c *Client) Generate(ctx context.Context, req llm.GenerateRequest) (*llm.GenerateResponse, error) {
	c.lastReq = req
	c.callCount++
	if c.err != nil {
		return nil, c.err
	}
	return c.resp, nil
}

// Name returns the provider name.
func (c *Client) Name() string { return c.name }

// LastReq returns the most recent GenerateRequest sent to this mock.
func (c *Client) LastReq() llm.GenerateRequest { return c.lastReq }

// CallCount returns the number of times Generate was called.
func (c *Client) CallCount() int { return c.callCount }

// Compile-time assertion that Client satisfies llm.Client
var _ llm.Client = (*Client)(nil)
