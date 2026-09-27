package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

const webhookTimeout = 10 * time.Second

type Payload struct {
	Status    string    `json:"status"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Service   string    `json:"service"`
}

func TriggerWebhook(ctx context.Context, webhookURL string, status string, message string, customHeaders map[string]string) error {
	if webhookURL == "" {
		slog.Debug("No webhook URL configured, skipping webhook notification")
		return nil
	}

	payload := Payload{
		Status:    status,
		Message:   message,
		Timestamp: time.Now(),
		Service:   "gitsaver",
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "gitsaver-webhook")

	for key, value := range customHeaders {
		req.Header.Set(key, value)
	}

	client := &http.Client{
		Timeout: webhookTimeout,
	}

	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook request: %w", err)
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode >= 200 && res.StatusCode < 300 {
		slog.Info("Webhook notification sent", "url", webhookURL, "status", res.StatusCode)
		return nil
	}

	body, _ := io.ReadAll(io.LimitReader(res.Body, 1024))
	return fmt.Errorf("webhook request failed with status code %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
}
