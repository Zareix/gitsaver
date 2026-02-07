package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type Payload struct {
	Status    string    `json:"status"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Service   string    `json:"service"`
}

func TriggerWebhook(webhookURL string, status string, message string, customHeaders map[string]string) error {
	if webhookURL == "" {
		log.Println("No webhook URL configured, skipping webhook notification")
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

	req, err := http.NewRequest("POST", webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "gitsaver-webhook")

	for key, value := range customHeaders {
		req.Header.Set(key, value)
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook request: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("Failed to close response body: %v", err)
		}
	}(resp.Body)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		log.Printf("Webhook notification sent successfully to %s (status: %d)", webhookURL, resp.StatusCode)
		return nil
	}

	return fmt.Errorf("webhook request failed with status code: %d", resp.StatusCode)
}
