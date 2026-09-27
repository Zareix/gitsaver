package config

import (
	"testing"
)

func TestIsTrueEnvVar(t *testing.T) {
	trues := []string{"true", "TRUE", "1", "yes", "Y", "on", "enabled", "enable"}
	for _, v := range trues {
		if !isTrueEnvVar(v) {
			t.Fatalf("isTrueEnvVar(%q) = false, want true", v)
		}
	}

	falses := []string{"false", "0", "no", "off", "", "disabled", "whatever"}
	for _, v := range falses {
		if isTrueEnvVar(v) {
			t.Fatalf("isTrueEnvVar(%q) = true, want false", v)
		}
	}
}

func TestParseWebhookHeaders(t *testing.T) {
	if len(parseWebhookHeaders("")) != 0 {
		t.Fatal("empty string should produce no headers")
	}

	headers := parseWebhookHeaders("Authorization: Bearer tok, X-Test : 1 , bad-format")
	if len(headers) != 2 {
		t.Fatalf("headers = %v, want 2 entries", headers)
	}
	if headers["Authorization"] != "Bearer tok" {
		t.Fatalf("Authorization = %q", headers["Authorization"])
	}
	if headers["X-Test"] != "1" {
		t.Fatalf("X-Test = %q", headers["X-Test"])
	}
}

func TestLoadConfigRequiresUsernameOrToken(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("GITHUB_USERNAME", "")

	if _, err := LoadConfig(); err == nil {
		t.Fatal("expected error when neither GITHUB_TOKEN nor GITHUB_USERNAME is set")
	}
}

func TestLoadConfigInvalidBackupMethod(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "tok")
	t.Setenv("GITHUB_BACKUP_METHOD", "rsync")

	if _, err := LoadConfig(); err == nil {
		t.Fatal("expected error for invalid GITHUB_BACKUP_METHOD")
	}
}

func TestLoadLogFormat(t *testing.T) {
	tests := []struct {
		name    string
		env     string
		want    LogFormat
		wantErr bool
	}{
		{"empty", "", LogFormatText, false},
		{"text", "text", LogFormatText, false},
		{"text uppercase", "TEXT", LogFormatText, false},
		{"json", "json", LogFormatJSON, false},
		{"json uppercase", "JSON", LogFormatJSON, false},
		{"invalid", "xml", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("LOG_FORMAT", tt.env)
			got, err := loadLogFormat()
			if (err != nil) != tt.wantErr {
				t.Fatalf("loadLogFormat(%q) error = %v, wantErr %v", tt.env, err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("loadLogFormat(%q) = %q, want %q", tt.env, got, tt.want)
			}
		})
	}
}

func TestLoadConfigDefaults(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "tok")
	for _, key := range []string{"GITHUB_BACKUP_METHOD", "DESTINATION_PATH", "GITHUB_CRON", "GITHUB_RUN_ON_STARTUP", "WEBHOOK_SUCCESS_URL", "WEBHOOK_FAILURE_URL", "WEBHOOK_HEADERS", "LOG_FORMAT"} {
		t.Setenv(key, "")
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.DestinationPath != "./output" {
		t.Fatalf("DestinationPath = %q", cfg.DestinationPath)
	}
	if cfg.Github.BackupMethod != Tarball {
		t.Fatalf("BackupMethod = %q, want tarball", cfg.Github.BackupMethod)
	}
	if cfg.LogFormat != LogFormatText {
		t.Fatalf("LogFormat = %q, want text", cfg.LogFormat)
	}
	if cfg.Github.Cron != "" {
		t.Fatalf("Cron = %q, want empty", cfg.Github.Cron)
	}
}
