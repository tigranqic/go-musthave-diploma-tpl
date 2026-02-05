package httpclient

import (
	"context"
	"errors"
	"net/http"
	"time"
)

var ErrRetryable = errors.New("retryable error")

type RetryClient struct {
	client     *http.Client
	MaxRetries int
	RetryDelay time.Duration
}

func NewRetryClient() *RetryClient {
	return &RetryClient{
		client:     &http.Client{Timeout: 5 * time.Second},
		MaxRetries: 3,
		RetryDelay: 200 * time.Millisecond,
	}
}

func (c *RetryClient) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	var lastErr error
	for i := 0; i <= c.MaxRetries; i++ {
		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = err
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			time.Sleep(c.RetryDelay)
			continue
		}
		if resp.StatusCode >= 500 {
			defer func() {
				_ = resp.Body.Close()
			}()
			lastErr = ErrRetryable
			time.Sleep(c.RetryDelay)
			continue
		}
		return resp, nil
	}
	return nil, lastErr
}
