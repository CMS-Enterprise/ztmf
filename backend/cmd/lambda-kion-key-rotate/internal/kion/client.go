// Package kion wraps the Kion App API Key rotation endpoint.
//
// The rotate call is a single idempotent operation from the client's point of
// view: the caller sends its current key, Kion invalidates it server-side and
// returns a fresh key. There is no explicit revoke-old step.
package kion

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const rotatePath = "/api/v3/app-api-key/rotate"

// Client calls the Kion App API Key rotation endpoint.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient returns a Client configured with a 30s request timeout that matches
// the local rotation tool. The httpClient is reused across calls.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

type rotateRequest struct {
	Key string `json:"key"`
}

type rotateResponse struct {
	Data struct {
		Key string `json:"key"`
	} `json:"data"`
}

// Rotate calls POST /api/v3/app-api-key/rotate with the current key as both the
// Bearer credential and the request body. On success it returns the new key.
//
// Retry policy: up to 3 attempts for transport errors and 5xx responses with
// exponential backoff (1s, 2s, 4s). Non-2xx responses in the 4xx range are not
// retried because the current key is already dead in those cases and retrying
// cannot recover it.
func (c *Client) Rotate(ctx context.Context, currentKey string) (string, error) {
	if currentKey == "" {
		return "", fmt.Errorf("kion: current key is empty")
	}

	body, err := json.Marshal(rotateRequest{Key: currentKey})
	if err != nil {
		return "", fmt.Errorf("kion: marshal request: %w", err)
	}

	var lastErr error
	for attempt := range 3 {
		if attempt > 0 {
			delay := time.Duration(1<<uint(attempt-1)) * time.Second
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(delay):
			}
		}

		newKey, retriable, err := c.rotateOnce(ctx, currentKey, body)
		if err == nil {
			return newKey, nil
		}
		lastErr = err
		if !retriable {
			return "", err
		}
	}

	return "", fmt.Errorf("kion: rotate failed after retries: %w", lastErr)
}

func (c *Client) rotateOnce(ctx context.Context, currentKey string, body []byte) (string, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+rotatePath, bytes.NewReader(body))
	if err != nil {
		return "", false, fmt.Errorf("kion: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+currentKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Transport errors (timeout, DNS, connection reset) are transient.
		return "", true, fmt.Errorf("kion: http do: %w", err)
	}
	defer resp.Body.Close()

	respBody, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return "", true, fmt.Errorf("kion: read response: %w", readErr)
	}

	if resp.StatusCode >= 500 {
		return "", true, fmt.Errorf("kion: status %d: %s", resp.StatusCode, truncate(string(respBody), 200))
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// 4xx: the current key is dead or invalid. No retry.
		return "", false, fmt.Errorf("kion: status %d: %s", resp.StatusCode, truncate(string(respBody), 200))
	}

	var parsed rotateResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", false, fmt.Errorf("kion: decode response: %w", err)
	}
	if parsed.Data.Key == "" {
		return "", false, fmt.Errorf("kion: response missing data.key")
	}
	return parsed.Data.Key, false, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
