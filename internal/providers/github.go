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

	"github.com/go-git/go-git/v6"
	gitConfig "github.com/go-git/go-git/v6/config"
	httpTransport "github.com/go-git/go-git/v6/plumbing/transport/http"
	"github.com/google/go-github/v81/github"
)

type GithubClient struct {
	ctx             context.Context
	isAuthenticated bool
	client          *github.Client
	username        string
}

func getClient(ctx context.Context, cfg *config.Config) (*GithubClient, error) {
	client := github.NewClient(nil)

	if cfg.Github.Token == "" {
		return &GithubClient{
			ctx:             ctx,
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
		ctx:             ctx,
		isAuthenticated: true,
		client:          client,
		username:        *user.Login,
	}, nil
}

func BackupGithubRepositories(ctx context.Context, cfg *config.Config) error {
	log.Printf("Starting GitHub repositories backup with %s method...", cfg.Github.BackupMethod)
	gClient, err := getClient(ctx, cfg)
	if err != nil {
		return err
	}

	repos := []*github.Repository{}
	if gClient.isAuthenticated {
		repos, err = getAuthenticatedRepositoriesList(gClient)
	} else {
		repos, err = getUnauthenticatedRepositoriesList(gClient)
	}
	if err != nil {
		return fmt.Errorf("Error fetching repositories list: %w", err)
	}

	var wg sync.WaitGroup
	for _, repo := range repos {
		wg.Add(1)
		go func(repo *github.Repository) {
			defer wg.Done()
			if shouldSkipRepository(*repo, cfg.Github, gClient.username) {
				return
			}

			switch cfg.Github.BackupMethod {
			case config.Tarball:
				err := downloadRepositoryTarball(gClient, *repo, cfg.DestinationPath, cfg.Github.ExtractTarballs)
				if err != nil {
					log.Println(fmt.Errorf("Error downloading repository %s: %w", *repo.Name, err).Error())
				}
			case config.Git:
				err := cloneRepository(*cfg, *repo.CloneURL, filepath.Join(cfg.DestinationPath, *repo.Owner.Login, *repo.Name))
				if err != nil {
					log.Println(fmt.Errorf("Error cloning repository %s: %w", *repo.Name, err).Error())
				}
			}
		}(repo)
	}
	wg.Wait()

	return nil
}

func getUnauthenticatedRepositoriesList(client *GithubClient) ([]*github.Repository, error) {
	if client.username == "" {
		return nil, fmt.Errorf("GitHub username is required for unauthenticated requests")
	}

	repos := []*github.Repository{}
	opt := &github.RepositoryListByUserOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	for {
		r, resp, err := client.client.Repositories.ListByUser(client.ctx, client.username, opt)
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

func getAuthenticatedRepositoriesList(client *GithubClient) ([]*github.Repository, error) {
	repos := []*github.Repository{}
	opt := &github.RepositoryListByAuthenticatedUserOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	for {
		r, resp, err := client.client.Repositories.ListByAuthenticatedUser(client.ctx, opt)
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

func shouldSkipRepository(repo github.Repository, cfg config.GithubProviderConfig, currentUsername string) bool {
	if !cfg.IncludeArchivedRepos && repo.GetArchived() {
		log.Printf("Skipping archived repository %s/%s", *repo.Owner.Login, *repo.Name)
		return true
	}

	if !cfg.IncludeForkedRepos && repo.GetFork() {
		log.Printf("Skipping forked repository %s/%s", *repo.Owner.Login, *repo.Name)
		return true
	}

	if !cfg.IncludeOtherUsersRepos && !strings.EqualFold(*repo.Owner.Login, currentUsername) {
		log.Printf("Skipping repository %s/%s owned by another user", *repo.Owner.Login, *repo.Name)
		return true
	}

	return false
}

func downloadRepositoryTarball(client *GithubClient, repo github.Repository, path string, shouldExtractTarball bool) error {
	log.Printf("Downloading tarball for repository %s/%s", *repo.Owner.Login, *repo.Name)

	link, _, err := client.client.Repositories.GetArchiveLink(client.ctx, *repo.Owner.Login, *repo.Name, github.Tarball, nil, 1)
	if err != nil {
		return err
	}

	resp, err := http.Get(link.String())
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if err := os.MkdirAll(filepath.Join(path, *repo.Owner.Login), 0755); err != nil {
		return err
	}

	filePath := filepath.Join(path, *repo.Owner.Login, *repo.Name+".tar.gz")
	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	if shouldExtractTarball {
		err = tarball.ExtractTarGz(filePath, filepath.Join(path, *repo.Owner.Login, *repo.Name))
	}

	log.Println("Successfully downloaded", *repo.Owner.Login+"/"+*repo.Name, "to", filePath)

	return nil
}

func cloneRepository(cfg config.Config, repoUrl, destPath string) error {
	auth := &httpTransport.BasicAuth{
		Username: "abc123",
		Password: cfg.Github.Token,
	}

	if _, err := os.Stat(destPath); err == nil {
		log.Printf("Repository %s already exists. Deleting it...", repoUrl)

		err = os.RemoveAll(destPath)
		if err != nil {
			return fmt.Errorf("Failed to remove existing repository: %w", err)
		}
	}

	log.Println("Cloning repository", repoUrl, "to", destPath)

	repo, err := git.PlainClone(destPath, &git.CloneOptions{
		URL:  repoUrl,
		Auth: auth,
	})
	if err != nil {
		return fmt.Errorf("Failed to clone repository: %w", err)
	}

	remote, err := repo.Remote("origin")
	if err != nil {
		return fmt.Errorf("Failed to get remote: %w", err)
	}

	if err := remote.Fetch(&git.FetchOptions{
		RefSpecs: []gitConfig.RefSpec{"refs/*:refs/*"},
		Auth:     auth,
	}); err != nil {
		if err != git.NoErrAlreadyUpToDate {
			return fmt.Errorf("Failed to fetch updates: %w", err)
		}
	}

	return err
}
