package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type BackupMethod string

const (
	Git     BackupMethod = "git"
	Tarball BackupMethod = "tarball"
)

type GithubProviderConfig struct {
	BackupMethod           BackupMethod
	RunOnStartup           bool
	Cron                   string
	Username               string
	Token                  string
	IncludeOtherUsersRepos bool
	IncludeForkedRepos     bool
	IncludeArchivedRepos   bool
	ExtractTarballs        bool
}

type Config struct {
	Github            GithubProviderConfig
	DestinationPath   string
	SuccessWebhookURL string
	FailureWebhookURL string
	WebhookHeaders    map[string]string
}

func LoadConfig() (Config, error) {
	if err := godotenv.Load(); err != nil {
		slog.Info("Could not load .env file, proceeding with environment variables")
	}

	destinationPath := os.Getenv("DESTINATION_PATH")
	if destinationPath == "" {
		destinationPath = "./output"
	}

	successWebhookURL := os.Getenv("WEBHOOK_SUCCESS_URL")
	failureWebhookURL := os.Getenv("WEBHOOK_FAILURE_URL")
	webhookHeaders := parseWebhookHeaders(os.Getenv("WEBHOOK_HEADERS"))

	githubConfig, err := loadGithubConfig()
	if err != nil {
		return Config{}, err
	}

	return Config{
		Github:            githubConfig,
		DestinationPath:   destinationPath,
		SuccessWebhookURL: successWebhookURL,
		WebhookHeaders:    webhookHeaders,
		FailureWebhookURL: failureWebhookURL,
	}, nil
}

func loadGithubConfig() (GithubProviderConfig, error) {
	backupMethodEnv := os.Getenv("GITHUB_BACKUP_METHOD")
	var backupMethod BackupMethod
	switch strings.ToLower(backupMethodEnv) {
	case "git":
		backupMethod = Git
	case "", "tarball":
		backupMethod = Tarball
	default:
		return GithubProviderConfig{}, fmt.Errorf("invalid GITHUB_BACKUP_METHOD %q, expected %q or %q", backupMethodEnv, Tarball, Git)
	}

	token := os.Getenv("GITHUB_TOKEN")
	username := os.Getenv("GITHUB_USERNAME")
	if token == "" && username == "" {
		return GithubProviderConfig{}, fmt.Errorf("GITHUB_TOKEN or GITHUB_USERNAME is required")
	}

	return GithubProviderConfig{
		BackupMethod:           backupMethod,
		RunOnStartup:           isTrueEnvVar(os.Getenv("GITHUB_RUN_ON_STARTUP")),
		Cron:                   os.Getenv("GITHUB_CRON"),
		Token:                  token,
		Username:               username,
		IncludeOtherUsersRepos: isTrueEnvVar(os.Getenv("GITHUB_INCLUDE_OTHER_USERS_REPOS")),
		IncludeForkedRepos:     isTrueEnvVar(os.Getenv("GITHUB_INCLUDE_FORKED_REPOS")),
		IncludeArchivedRepos:   isTrueEnvVar(os.Getenv("GITHUB_INCLUDE_ARCHIVED_REPOS")),
		ExtractTarballs:        isTrueEnvVar(os.Getenv("GITHUB_EXTRACT_TARBALLS")),
	}, nil
}

func isTrueEnvVar(value string) bool {
	val := strings.ToLower(value)
	return val == "true" || val == "1" || val == "yes" || val == "y" || val == "on" || val == "enabled" || val == "enable"
}

func parseWebhookHeaders(headersStr string) map[string]string {
	headers := make(map[string]string)
	if headersStr == "" {
		return headers
	}

	pairs := strings.Split(headersStr, ",")
	for _, pair := range pairs {
		parts := strings.SplitN(strings.TrimSpace(pair), ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			if key != "" {
				headers[key] = value
			}
		}
	}
	return headers
}
