package webhook

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTriggerWebhookSuccess(t *testing.T) {
	var gotAuth string
	var gotService string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotService = r.Header.Get("X-Service-Check")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	err := TriggerWebhook(context.Background(), server.URL, "success", "done", map[string]string{
		"Authorization":   "Bearer tok",
		"X-Service-Check": "1",
	})
	if err != nil {
		t.Fatalf("trigger webhook: %v", err)
	}
	if gotAuth != "Bearer tok" {
		t.Fatalf("authorization = %q, want %q", gotAuth, "Bearer tok")
	}
	if gotService != "1" {
		t.Fatalf("custom header = %q", gotService)
	}
}

func TestTriggerWebhookServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer server.Close()

	err := TriggerWebhook(context.Background(), server.URL, "success", "done", nil)
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Fatalf("expected status code in error, got: %v", err)
	}
}

func TestTriggerWebhookEmptyURL(t *testing.T) {
	if err := TriggerWebhook(context.Background(), "", "success", "done", nil); err != nil {
		t.Fatalf("empty URL should be a no-op, got: %v", err)
	}
}
