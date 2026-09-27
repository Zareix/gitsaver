package providers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gitsaver/internal/config"
	"gitsaver/internal/tarball"

	"github.com/go-git/go-git/v6"
	gitClient "github.com/go-git/go-git/v6/plumbing/client"
	httpTransport "github.com/go-git/go-git/v6/plumbing/transport/http"
	"github.com/google/go-github/v83/github"
	"golang.org/x/sync/errgroup"
)

const maxConcurrentBackups = 8

type GithubClient struct {
	ctx             context.Context
	isAuthenticated bool
	client          *github.Client
	username        string
}

func getClient(ctx context.Context, cfg config.Config) (*GithubClient, error) {
	githubClient := github.NewClient(nil)

	if cfg.Github.Token == "" {
		return &GithubClient{
			ctx:             ctx,
			isAuthenticated: false,
			client:          githubClient,
			username:        cfg.Github.Username,
		}, nil
	}

	slog.Info("Authenticating with GITHUB_TOKEN")
	githubClient = githubClient.WithAuthToken(cfg.Github.Token)

	user, _, err := githubClient.Users.Get(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("failed to authenticate with GitHub: %w", err)
	}
	slog.Info("Authenticated", "user", user.GetLogin())

	return &GithubClient{
		ctx:             ctx,
		isAuthenticated: true,
		client:          githubClient,
		username:        user.GetLogin(),
	}, nil
}

func BackupGithubRepositories(ctx context.Context, cfg config.Config) error {
	slog.Info("Starting GitHub repositories backup", "method", cfg.Github.BackupMethod)

	gClient, err := getClient(ctx, cfg)
	if err != nil {
		return err
	}

	var repos []*github.Repository
	if gClient.isAuthenticated {
		repos, err = getAuthenticatedRepositoriesList(gClient)
	} else {
		repos, err = getUnauthenticatedRepositoriesList(gClient)
	}
	if err != nil {
		return fmt.Errorf("error fetching repositories list: %w", err)
	}
	slog.Info("Repositories to backup", "count", len(repos))

	var (
		mu   sync.Mutex
		errs []error
	)
	var group errgroup.Group
	group.SetLimit(maxConcurrentBackups)

	for _, repo := range repos {
		group.Go(func() error {
			if err := backupRepository(ctx, cfg, gClient, repo); err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("%s: %w", repo.GetFullName(), err))
				mu.Unlock()
			}
			return nil
		})
	}
	_ = group.Wait()

	return errors.Join(errs...)
}

func backupRepository(ctx context.Context, cfg config.Config, gClient *GithubClient, repo *github.Repository) error {
	if shouldSkipRepository(repo, cfg.Github, gClient.username) {
		return nil
	}

	switch cfg.Github.BackupMethod {
	case config.Tarball:
		return downloadRepositoryTarball(ctx, gClient, repo, cfg.DestinationPath, cfg.Github.ExtractTarballs)
	case config.Git:
		destPath := filepath.Join(cfg.DestinationPath, repo.GetOwner().GetLogin(), repo.GetName())
		return cloneRepository(ctx, cfg, repo.GetCloneURL(), destPath)
	default:
		return fmt.Errorf("unsupported backup method %q", cfg.Github.BackupMethod)
	}
}

func getUnauthenticatedRepositoriesList(gClient *GithubClient) ([]*github.Repository, error) {
	if gClient.username == "" {
		return nil, errors.New("GITHUB_USERNAME is required for unauthenticated requests")
	}

	var repos []*github.Repository
	opt := &github.RepositoryListByUserOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	for {
		r, resp, err := gClient.client.Repositories.ListByUser(gClient.ctx, gClient.username, opt)
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

func getAuthenticatedRepositoriesList(gClient *GithubClient) ([]*github.Repository, error) {
	var repos []*github.Repository
	opt := &github.RepositoryListByAuthenticatedUserOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	for {
		r, resp, err := gClient.client.Repositories.ListByAuthenticatedUser(gClient.ctx, opt)
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

func shouldSkipRepository(repo *github.Repository, cfg config.GithubProviderConfig, currentUsername string) bool {
	if !cfg.IncludeArchivedRepos && repo.GetArchived() {
		slog.Debug("Skipping archived repository", "repository", repo.GetFullName())
		return true
	}

	if !cfg.IncludeForkedRepos && repo.GetFork() {
		slog.Debug("Skipping forked repository", "repository", repo.GetFullName())
		return true
	}

	if !cfg.IncludeOtherUsersRepos && !strings.EqualFold(repo.GetOwner().GetLogin(), currentUsername) {
		slog.Debug("Skipping repository owned by another user", "repository", repo.GetFullName())
		return true
	}

	return false
}

func downloadRepositoryTarball(ctx context.Context, gClient *GithubClient, repo *github.Repository, path string, shouldExtractTarball bool) error {
	owner, name := repo.GetOwner().GetLogin(), repo.GetName()
	slog.Info("Downloading tarball", "repository", owner+"/"+name)

	link, _, err := gClient.client.Repositories.GetArchiveLink(gClient.ctx, owner, name, github.Tarball, nil, 1)
	if err != nil {
		return fmt.Errorf("failed to get archive link: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to create download request: %w", err)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download archive: %w", err)
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 1024))
		return fmt.Errorf("archive download returned %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}

	if err := os.MkdirAll(filepath.Join(path, owner), 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	filePath := filepath.Join(path, owner, name+".tar.gz")
	out, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() { _ = out.Close() }()

	if _, err := io.Copy(out, res.Body); err != nil {
		return fmt.Errorf("failed to write archive: %w", err)
	}

	if shouldExtractTarball {
		if err := tarball.ExtractTarGz(filePath, filepath.Join(path, owner, name)); err != nil {
			return fmt.Errorf("failed to extract tarball: %w", err)
		}
	}

	slog.Info("Tarball downloaded", "repository", owner+"/"+name, "path", filePath)

	return nil
}

func cloneRepository(ctx context.Context, cfg config.Config, repoURL, destPath string) error {
	var clientOptions []gitClient.Option
	if cfg.Github.Token != "" {
		clientOptions = append(clientOptions, gitClient.WithHTTPAuth(&httpTransport.BasicAuth{
			Username: "abc123",
			Password: cfg.Github.Token,
		}))
	}

	if _, err := os.Stat(destPath); err == nil {
		slog.Info("Repository already exists, deleting it", "path", destPath)
		if err := os.RemoveAll(destPath); err != nil {
			return fmt.Errorf("failed to remove existing repository: %w", err)
		}
	}

	slog.Info("Cloning repository", "url", repoURL, "path", destPath)

	repo, err := git.PlainCloneContext(ctx, destPath, &git.CloneOptions{
		URL:           repoURL,
		ClientOptions: clientOptions,
	})
	if err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}
	defer func() { _ = repo.Close() }()

	return nil
}
