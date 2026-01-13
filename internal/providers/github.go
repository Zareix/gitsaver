package providers

import (
	"context"
	"fmt"
	"gitsaver/internal/config"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/go-github/v81/github"
)

type GithubClient struct {
	isAuthenticated bool
	client          *github.Client
	username        string
}

func getClient(ctx context.Context, cfg *config.Config) *GithubClient {
	client := github.NewClient(nil)

	if cfg.Github.Token == "" {
		return &GithubClient{
			isAuthenticated: false,
			client:          client,
			username:        cfg.Github.Username,
		}
	}

	println("Login with GITHUB_TOKEN")
	client = client.WithAuthToken(cfg.Github.Token)

	user, _, err := client.Users.Get(ctx, "")
	if err != nil {
		panic(err)
	}
	println("Logged in as:", *user.Login)
	return &GithubClient{
		isAuthenticated: true,
		client:          client,
		username:        *user.Login,
	}
}

func BackupGithubRepositories(ctx context.Context, cfg *config.Config) {
	gClient := getClient(ctx, cfg)

	repos := []*github.Repository{}
	var err error
	if gClient.isAuthenticated {
		repos, err = getAuthenticatedRepositoriesList(ctx, gClient)
	} else {
		repos, err = getUnauthenticatedRepositoriesList(ctx, gClient)
	}
	if err != nil {
		panic(err)
	}

	// var wg sync.WaitGroup
	// errChan := make(chan error, len(repos))

	// for _, repo := range repos {
	// 	wg.Add(1)
	// 	go func(repo *github.Repository) {
	// 		defer wg.Done()
	// 		err := downloadRepository(ctx, gClient.client, *repo, cfg.DestinationPath)
	// 		if err != nil {
	// 			errChan <- fmt.Errorf("error downloading repository %s: %w", *repo.Name, err)
	// 		}
	// 	}(repo)
	// }

	// go func() {
	// 	wg.Wait()
	// 	close(errChan)
	// }()

	// for err := range errChan {
	// 	println(err.Error())
	// }

	for _, repo := range repos {
		if !cfg.Github.IncludeOtherUsersRepos && !strings.EqualFold(*repo.Owner.Login, gClient.username) {
			println(fmt.Sprintf("Skipping repository %s/%s", *repo.Owner.Login, *repo.Name))
			continue
		}

		if !cfg.Github.IncludeForkedRepos && repo.GetFork() {
			println(fmt.Sprintf("Skipping forked repository %s/%s", *repo.Owner.Login, *repo.Name))
			continue
		}

		err = downloadRepository(ctx, gClient.client, *repo, cfg.DestinationPath)
		if err != nil {
			println(fmt.Errorf("error downloading repository %s: %w", *repo.Name, err).Error())
		}
	}
}

func getUnauthenticatedRepositoriesList(ctx context.Context, client *GithubClient) ([]*github.Repository, error) {
	if client.username == "" {
		panic("GITHUB_USERNAME environment variable is required for unauthenticated access")
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

func downloadRepository(ctx context.Context, client *github.Client, repo github.Repository, path string) error {
	link, _, err := client.Repositories.GetArchiveLink(ctx, *repo.Owner.Login, *repo.Name, github.Tarball, nil, 1)
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

	out, err := os.Create(filepath.Join(path, *repo.Owner.Login, *repo.Name+".tar.gz"))
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	println("Successfully downloaded", *repo.Owner.Login+"/"+*repo.Name, "to", path)

	return nil
}
