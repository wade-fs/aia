package http

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// Client wraps net/http with rate limiting, timeouts, and User-Agent compliance.
type Client struct {
	httpClient *http.Client
	semaphore  chan struct{}
	userAgent  string
	mu         sync.Mutex
}

func NewClient(timeout time.Duration, maxConcurrent int, userAgent string) *Client {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	if maxConcurrent <= 0 {
		maxConcurrent = 5
	}
	if userAgent == "" {
		userAgent = "AIA-Personal-Agent/1.0 (Data Collector; +https://github.com/wade/aia)"
	}

	return &Client{
		httpClient: &http.Client{Timeout: timeout},
		semaphore:  make(chan struct{}, maxConcurrent),
		userAgent:  userAgent,
	}
}

// Get performs an HTTP GET request with rate limiting and context awareness.
func (c *Client) Get(ctx context.Context, url string) ([]byte, int, error) {
	select {
	case c.semaphore <- struct{}{}:
		defer func() { <-c.semaphore }()
	case <-ctx.Done():
		return nil, 0, ctx.Err()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json, text/plain, */*")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("failed to read response body: %w", err)
	}

	return body, resp.StatusCode, nil
}
