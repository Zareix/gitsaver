package gitsaver

import (
	"context"
	"gitsaver/internal/providers"
)

func Run() {
	ctx := context.Background()

	providers.BackupGithubRepositories(ctx)
}
