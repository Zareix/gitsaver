package main

import (
	"context"
	"fmt"
	"gitsaver/internal/config"
	"gitsaver/internal/providers"
	"gitsaver/internal/webhook"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-co-op/gocron-ui/server"
	"github.com/go-co-op/gocron/v2"
)

const Version = "1.3.0"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "health" {
		port := os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}
		resp, err := http.Get("http://localhost:" + port + "/api/jobs")
		if err != nil || resp.StatusCode >= 400 {
			os.Exit(1)
		}
		os.Exit(0)
	}

	ctx := context.Background()
	cfg := config.LoadConfig()

	scheduler, err := gocron.NewScheduler()
	if err != nil {
		log.Fatal(err)
	}

	if cfg.Github.Cron != "" {
		_, err = scheduler.NewJob(
			gocron.CronJob(cfg.Github.Cron, false),
			gocron.NewTask(runGithubBackupJob, ctx, cfg),
			gocron.WithName("GitHub Backup Job"),
		)
		if err != nil {
			log.Fatal(err)
		}
		slog.Info("Scheduled GitHub backup job with cron", "cron", cfg.Github.Cron)
	}

	if cfg.Github.RunOnStartup {
		slog.Info("Running GitHub backup job on startup")
		go func() {
			runGithubBackupJob(ctx, cfg)
		}()
	}

	if len(scheduler.Jobs()) == 0 {
		slog.Info("No backup jobs scheduled. Exiting.")
		return
	}

	scheduler.Start()

	srv := server.NewServer(scheduler, cfg.Port, server.WithTitle("Gitsaver Scheduler"))
	slog.Info("Gitsaver is running", "version", Version, "port", cfg.Port)
	err = http.ListenAndServe(fmt.Sprintf(":%d", cfg.Port), srv.Router)
	if err != nil {
		slog.Error("Failed to start HTTP server:", err)
		return
	}
}

func runGithubBackupJob(ctx context.Context, cfg config.Config) {
	err := providers.BackupGithubRepositories(ctx, cfg)
	if err != nil {
		if webhookErr := webhook.TriggerWebhook(cfg.FailureWebhookURL, "failure", fmt.Sprintf("GitHub backup failed: %v", err), cfg.WebhookHeaders); webhookErr != nil {
			slog.Warn("Failed to trigger failure webhook:", webhookErr)
		}
		slog.Error("GitHub backup job failed:", err)
		return
	}

	if err := webhook.TriggerWebhook(cfg.SuccessWebhookURL, "success", "GitHub backup completed successfully", cfg.WebhookHeaders); err != nil {
		slog.Warn("Failed to trigger success webhook:", err)
	}
	slog.Info("GitHub backup job completed")
}
