package gitsaver

import (
	"context"
	"gitsaver/internal/config"
	"gitsaver/internal/providers"
)

func Run() {
	ctx := context.Background()
	cfg := config.LoadConfig()

	providers.BackupGithubRepositories(ctx, cfg)
}
