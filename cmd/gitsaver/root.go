package gitsaver

import (
	"context"
	"fmt"
	"gitsaver/internal/config"
	"gitsaver/internal/providers"
	"log"
	"net/http"

	"github.com/go-co-op/gocron-ui/server"
	"github.com/go-co-op/gocron/v2"
)

func Run() {
	ctx := context.Background()
	cfg := config.LoadConfig()

	scheduler, err := gocron.NewScheduler()
	if err != nil {
		log.Fatal(err)
	}

	if cfg.Github.Cron != "" {
		_, err = scheduler.NewJob(
			gocron.CronJob(cfg.Github.Cron, false),
			gocron.NewTask(func() {
				err := providers.BackupGithubRepositories(ctx, cfg)
				if err != nil {
					log.Println("GitHub backup job failed:", err)
				}
			}),
			gocron.WithName("GitHub Backup Job"),
		)
		if err != nil {
			log.Fatal(err)
		}
		log.Println("Scheduled GitHub backup job with cron:", cfg.Github.Cron)
	}

	if cfg.Github.RunOnStartup {
		log.Println("Running GitHub backup job on startup")
		go func() {
			err := providers.BackupGithubRepositories(ctx, cfg)
			if err != nil {
				log.Fatal("GitHub backup job failed:", err)
			}
		}()
	}

	if len(scheduler.Jobs()) == 0 {
		log.Println("No backup jobs scheduled. Exiting.")
		return
	}

	scheduler.Start()

	srv := server.NewServer(scheduler, cfg.Port, server.WithTitle("Gitsaver Scheduler"))
	log.Printf("Gitsaver available at http://localhost:%d", cfg.Port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", cfg.Port), srv.Router))
}
