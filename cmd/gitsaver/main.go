package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"gitsaver/internal/config"
	"gitsaver/internal/providers"
	"gitsaver/internal/webhook"

	"github.com/robfig/cron/v3"
)

const Version = "1.4.0"

const jobRunTimeout = 30 * time.Minute

var backupMu sync.Mutex

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	if cfg.Github.Cron == "" {
		slog.Info("No CRON configured, exiting")
		return
	}

	scheduler := cron.New(
		cron.WithChain(cron.Recover(cron.DefaultLogger), cron.SkipIfStillRunning(cron.DefaultLogger)),
	)

	_, err = scheduler.AddFunc(cfg.Github.Cron, func() {
		runBackupJob(ctx, cfg)
	})
	if err != nil {
		slog.Error("Failed to schedule backup job", "error", err)
		os.Exit(1)
	}

	scheduler.Start()
	slog.Info("Scheduler started", "cron", cfg.Github.Cron, "version", Version)

	if cfg.Github.RunOnStartup {
		slog.Info("Running GitHub backup job on startup")
		go runBackupJob(ctx, cfg)
	}

	<-ctx.Done()
	slog.Info("Shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	stopCtx := scheduler.Stop()
	select {
	case <-stopCtx.Done():
	case <-shutdownCtx.Done():
		slog.Warn("Shutdown timeout reached, stopping scheduler")
	}
}

func runBackupJob(ctx context.Context, cfg config.Config) {
	backupMu.Lock()
	defer backupMu.Unlock()

	slog.Info("Running GitHub backup job...")

	jobCtx, cancel := context.WithTimeout(ctx, jobRunTimeout)
	defer cancel()

	if err := providers.BackupGithubRepositories(jobCtx, cfg); err != nil {
		if webhookErr := webhook.TriggerWebhook(ctx, cfg.FailureWebhookURL, "failure", fmt.Sprintf("GitHub backup failed: %v", err), cfg.WebhookHeaders); webhookErr != nil {
			slog.Warn("Failed to trigger failure webhook", "error", webhookErr)
		}
		slog.Error("GitHub backup job failed", "error", err)
		return
	}

	if err := webhook.TriggerWebhook(ctx, cfg.SuccessWebhookURL, "success", "GitHub backup completed successfully", cfg.WebhookHeaders); err != nil {
		slog.Warn("Failed to trigger success webhook", "error", err)
	}
	slog.Info("GitHub backup job completed")
}
