package config

import "os"

type GithubProviderConfig struct {
	Username               string
	Token                  string
	IncludeOtherUsersRepos bool
	IncludeForkedRepos     bool
}

type Config struct {
	Github          GithubProviderConfig
	DestinationPath string
}

func LoadConfig() *Config {
	destinationPath, destinationPathExists := os.LookupEnv("DESTINATION_PATH")
	if !destinationPathExists {
		destinationPath = "./output"
	}

	return &Config{
		Github: GithubProviderConfig{
			Token:                  os.Getenv("GITHUB_TOKEN"),
			Username:               os.Getenv("GITHUB_USERNAME"),
			IncludeOtherUsersRepos: os.Getenv("GITHUB_INCLUDE_OTHER_USERS_REPOS") == "true",
			IncludeForkedRepos:     os.Getenv("GITHUB_INCLUDE_FORKED_REPOS") == "true",
		},
		DestinationPath: destinationPath,
	}
}
