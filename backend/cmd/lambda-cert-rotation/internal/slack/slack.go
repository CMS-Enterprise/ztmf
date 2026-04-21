package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const disabledSentinel = "DISABLED"

func IsDisabledWebhookValue(v string) bool {
	return strings.EqualFold(strings.TrimSpace(v), disabledSentinel)
}

type Client struct {
	WebhookURL string
	HTTP       *http.Client
}

func (c Client) PostText(ctx context.Context, text string) error {
	if IsDisabledWebhookValue(c.WebhookURL) || strings.TrimSpace(c.WebhookURL) == "" {
		return nil
	}
	hc := c.HTTP
	if hc == nil {
		hc = &http.Client{Timeout: 10 * time.Second}
	}

	payload := map[string]string{"text": text}
	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal slack payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.WebhookURL, bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("create slack request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := hc.Do(req)
	if err != nil {
		return fmt.Errorf("post slack webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("slack webhook http %d", resp.StatusCode)
	}
	return nil
}
