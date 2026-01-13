package providers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/google/go-github/v81/github"
)

type GithubClient struct {
	isAuthenticated bool
	client          *github.Client
}

func getClient(ctx context.Context) *GithubClient {
	client := github.NewClient(nil)

	if token, tokenExists := os.LookupEnv("GITHUB_TOKEN"); tokenExists {
		println("Login with GITHUB_TOKEN")
		client = client.WithAuthToken(token)

		user, _, err := client.Users.Get(ctx, "")
		if err != nil {
			panic(err)
		}
		println("Logged in as:", *user.Login)
		return &GithubClient{
			isAuthenticated: tokenExists,
			client:          client,
		}
	}
	return &GithubClient{
		isAuthenticated: false,
		client:          client,
	}
}

func BackupGithubRepositories(ctx context.Context) {
	gClient := getClient(ctx)

	repos := []*github.Repository{}
	var err error
	if gClient.isAuthenticated {
		repos, err = getAuthenticatedRepositoriesList(ctx, gClient.client)
	} else {
		username, usernameExists := os.LookupEnv("GITHUB_USERNAME")
		if !usernameExists {
			panic("GITHUB_USERNAME environment variable is required for unauthenticated access")
		}
		repos, err = getUnauthenticatedRepositoriesList(ctx, gClient.client, username)
	}
	if err != nil {
		panic(err)
	}

	destinationPath, destinationPathExists := os.LookupEnv("GITSAVER_DESTINATION_PATH")
	if !destinationPathExists {
		destinationPath = "./output/github"
	}
	// var wg sync.WaitGroup
	// errChan := make(chan error, len(repos))

	// for _, repo := range repos {
	// 	wg.Add(1)
	// 	go func(repo *github.Repository) {
	// 		defer wg.Done()
	// 		err := downloadRepository(repo, destinationPath)
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
	//
	for _, repo := range repos {
		err := downloadRepository(*repo, destinationPath)
		if err != nil {
			println(fmt.Errorf("error downloading repository %s: %w", *repo.Name, err).Error())
		}
	}

}

func getUnauthenticatedRepositoriesList(ctx context.Context, client *github.Client, username string) ([]*github.Repository, error) {
	repos := []*github.Repository{}
	opt := &github.RepositoryListByUserOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	for {
		r, resp, err := client.Repositories.ListByUser(ctx, username, opt)
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

func getAuthenticatedRepositoriesList(ctx context.Context, client *github.Client) ([]*github.Repository, error) {
	repos := []*github.Repository{}
	opt := &github.RepositoryListByAuthenticatedUserOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	for {
		r, resp, err := client.Repositories.ListByAuthenticatedUser(ctx, opt)
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

func downloadRepository(repo github.Repository, path string) error {
	resp, err := http.Get(*repo.ArchiveURL)
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
