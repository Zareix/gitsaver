package config

import (
	"log"
	"os"
	"strconv"
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
	Port              int
	SuccessWebhookURL string
	FailureWebhookURL string
	WebhookHeaders    map[string]string
}

func LoadConfig() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("Could not load .env file, proceeding with environment variables")
	}

	destinationPath, destinationPathExists := os.LookupEnv("DESTINATION_PATH")
	if !destinationPathExists {
		destinationPath = "./output"
	}

	portEnv := os.Getenv("PORT")
	if portEnv == "" {
		portEnv = "8080"
	}
	port, err := strconv.Atoi(portEnv)
	if err != nil {
		log.Fatalf("Invalid PORT value: %v", err)
	}

	successWebhookURL := os.Getenv("WEBHOOK_SUCCESS_URL")
	failureWebhookURL := os.Getenv("WEBHOOK_FAILURE_URL")
	webhookHeaders := parseWebhookHeaders(os.Getenv("WEBHOOK_HEADERS"))

	return &Config{
		Github:            loadGithubConfig(),
		DestinationPath:   destinationPath,
		SuccessWebhookURL: successWebhookURL,
		WebhookHeaders:    webhookHeaders,
		FailureWebhookURL: failureWebhookURL,
		Port:              port,
	}
}

func loadGithubConfig() GithubProviderConfig {
	backupMethodEnv := os.Getenv("GITHUB_BACKUP_METHOD")
	var backupMethod BackupMethod
	switch strings.ToLower(backupMethodEnv) {
	case "git":
		backupMethod = Git
	default:
		backupMethod = Tarball
	}

	return GithubProviderConfig{
		BackupMethod:           backupMethod,
		RunOnStartup:           isTrueEnvVar(os.Getenv("GITHUB_RUN_ON_STARTUP")),
		Cron:                   os.Getenv("GITHUB_CRON"),
		Token:                  os.Getenv("GITHUB_TOKEN"),
		Username:               os.Getenv("GITHUB_USERNAME"),
		IncludeOtherUsersRepos: isTrueEnvVar(os.Getenv("GITHUB_INCLUDE_OTHER_USERS_REPOS")),
		IncludeForkedRepos:     isTrueEnvVar(os.Getenv("GITHUB_INCLUDE_FORKED_REPOS")),
		IncludeArchivedRepos:   isTrueEnvVar(os.Getenv("GITHUB_INCLUDE_ARCHIVED_REPOS")),
		ExtractTarballs:        isTrueEnvVar(os.Getenv("GITHUB_EXTRACT_TARBALLS")),
	}
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
