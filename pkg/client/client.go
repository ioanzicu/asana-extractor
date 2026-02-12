package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ioanzicu/asana-extractor/pkg/ratelimit"
	"github.com/ioanzicu/asana-extractor/pkg/retry"
)

// Client wraps http.Client with rate limiting and retry logic
type Client struct {
	httpClient  *http.Client
	rateLimiter *ratelimit.Limiter
	retryConfig retry.Config
	token       string
}

// Config holds client configuration
type Config struct {
	Token           string
	RateLimitConfig ratelimit.Config
	RetryConfig     retry.Config
	Timeout         time.Duration
}

// New creates a new HTTP client with rate limiting and retry logic
func New(cfg Config) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		rateLimiter: ratelimit.NewLimiter(cfg.RateLimitConfig),
		retryConfig: cfg.RetryConfig,
		token:       cfg.Token,
	}
}

// Do executes an HTTP request with rate limiting and retry logic
func (c *Client) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	// Determine request type for rate limiting
	reqType := ratelimit.RequestTypeRead
	if req.Method != http.MethodGet && req.Method != http.MethodHead {
		reqType = ratelimit.RequestTypeWrite
	}

	// Acquire rate limit slot
	if err := c.rateLimiter.Acquire(ctx, reqType); err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
	}
	defer c.rateLimiter.Release(reqType)

	// Add authentication header
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")

	// Execute with retry logic
	resp, err := retry.Do(ctx, c.retryConfig, func() (*http.Response, error) {
		// Clone the request for retry attempts
		reqClone := req.Clone(ctx)
		return c.httpClient.Do(reqClone)
	})

	return resp, err
}

// Get performs a GET request
func (c *Client) Get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	return c.Do(ctx, req)
}

// GetBody performs a GET request and returns the response body as bytes
func (c *Client) GetBody(ctx context.Context, url string) ([]byte, error) {
	resp, err := c.Get(ctx, url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
}
