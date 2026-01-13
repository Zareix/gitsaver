package providers

import (
	"context"
	"fmt"
	"gitsaver/internal/config"
	"gitsaver/internal/tarball"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/google/go-github/v81/github"
)

type GithubClient struct {
	isAuthenticated bool
	client          *github.Client
	username        string
}

func getClient(ctx context.Context, cfg *config.Config) (*GithubClient, error) {
	client := github.NewClient(nil)

	if cfg.Github.Token == "" {
		return &GithubClient{
			isAuthenticated: false,
			client:          client,
			username:        cfg.Github.Username,
		}, nil
	}

	log.Println("Login with GITHUB_TOKEN")
	client = client.WithAuthToken(cfg.Github.Token)

	user, _, err := client.Users.Get(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("Failed to authenticate with GitHub: %w", err)
	}
	log.Println("Logged in as:", *user.Login)

	return &GithubClient{
		isAuthenticated: true,
		client:          client,
		username:        *user.Login,
	}, nil
}

func BackupGithubRepositories(ctx context.Context, cfg *config.Config) error {
	gClient, err := getClient(ctx, cfg)
	if err != nil {
		return err
	}

	repos := []*github.Repository{}
	if gClient.isAuthenticated {
		repos, err = getAuthenticatedRepositoriesList(ctx, gClient)
	} else {
		repos, err = getUnauthenticatedRepositoriesList(ctx, gClient)
	}
	if err != nil {
		return fmt.Errorf("Error fetching repositories list: %w", err)
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(repos))

	for _, repo := range repos {
		wg.Add(1)
		go func(repo *github.Repository) {
			defer wg.Done()
			if !cfg.Github.IncludeOtherUsersRepos && !strings.EqualFold(*repo.Owner.Login, gClient.username) {
				log.Printf("Skipping repository %s/%s", *repo.Owner.Login, *repo.Name)
				return
			}

			if !cfg.Github.IncludeForkedRepos && repo.GetFork() {
				log.Printf("Skipping forked repository %s/%s", *repo.Owner.Login, *repo.Name)
				return
			}

			fileName, err := downloadRepositoryTarball(ctx, gClient.client, *repo, cfg.DestinationPath)
			if err != nil {
				log.Println(fmt.Errorf("error downloading repository %s: %w", *repo.Name, err).Error())
			}
			if cfg.Github.ExtractTarballs {
				err = tarball.ExtractTarGz(fileName, filepath.Join(cfg.DestinationPath, *repo.Owner.Login, *repo.Name))
			}
		}(repo)
	}

	go func() {
		wg.Wait()
		close(errChan)
	}()

	for err := range errChan {
		if err != nil {
			log.Println("Error:", err)
		}
	}

	return nil
}

func getUnauthenticatedRepositoriesList(ctx context.Context, client *GithubClient) ([]*github.Repository, error) {
	if client.username == "" {
		return nil, fmt.Errorf("GitHub username is required for unauthenticated requests")
	}

	repos := []*github.Repository{}
	opt := &github.RepositoryListByUserOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	for {
		r, resp, err := client.client.Repositories.ListByUser(ctx, client.username, opt)
		if err != nil {
			return nil, err
		}
		repos = append(repos, r...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	return repos, nil
}

func getAuthenticatedRepositoriesList(ctx context.Context, client *GithubClient) ([]*github.Repository, error) {
	repos := []*github.Repository{}
	opt := &github.RepositoryListByAuthenticatedUserOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	for {
		r, resp, err := client.client.Repositories.ListByAuthenticatedUser(ctx, opt)
		if err != nil {
			return nil, err
		}
		repos = append(repos, r...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	return repos, nil
}

func downloadRepositoryTarball(ctx context.Context, client *github.Client, repo github.Repository, path string) (string, error) {
	link, _, err := client.Repositories.GetArchiveLink(ctx, *repo.Owner.Login, *repo.Name, github.Tarball, nil, 1)
	if err != nil {
		return "", err
	}

	resp, err := http.Get(link.String())
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if err := os.MkdirAll(filepath.Join(path, *repo.Owner.Login), 0755); err != nil {
		return "", err
	}

	filePath := filepath.Join(path, *repo.Owner.Login, *repo.Name+".tar.gz")
	out, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", err
	}

	log.Println("Successfully downloaded", *repo.Owner.Login+"/"+*repo.Name, "to", filePath)

	return filePath, nil
}
