package config

import (
	"errors"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type GithubProviderConfig struct {
	RunOnStartup           bool
	Cron                   string
	Username               string
	Token                  string
	IncludeOtherUsersRepos bool
	IncludeForkedRepos     bool
	ExtractTarballs        bool
}

type Config struct {
	Github          GithubProviderConfig
	DestinationPath string
	Port            int
}

func LoadConfig() *Config {
	if _, err := os.Stat(".env"); !errors.Is(err, os.ErrNotExist) {
		err := godotenv.Load()
		if err != nil {
			log.Fatal("Error loading .env file")
		}
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

	return &Config{
		Github:          loadGithubConfig(),
		DestinationPath: destinationPath,
		Port:            port,
	}
}

func loadGithubConfig() GithubProviderConfig {
	return GithubProviderConfig{
		RunOnStartup:           isTrueEnvVar(os.Getenv("GITHUB_RUN_ON_STARTUP")),
		Cron:                   os.Getenv("GITHUB_CRON"),
		Token:                  os.Getenv("GITHUB_TOKEN"),
		Username:               os.Getenv("GITHUB_USERNAME"),
		IncludeOtherUsersRepos: isTrueEnvVar(os.Getenv("GITHUB_INCLUDE_OTHER_USERS_REPOS")),
		IncludeForkedRepos:     isTrueEnvVar(os.Getenv("GITHUB_INCLUDE_FORKED_REPOS")),
		ExtractTarballs:        isTrueEnvVar(os.Getenv("GITHUB_EXTRACT_TARBALLS")),
	}
}

func isTrueEnvVar(value string) bool {
	val := strings.ToLower(value)
	return val == "true" || val == "1" || val == "yes" || val == "y" || val == "on" || val == "enabled" || val == "enable"
}
