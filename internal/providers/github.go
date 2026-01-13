package providers

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

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
		err := getContents(ctx, gClient.client, *repo.Owner.Login, *repo.Name, "", filepath.Join(destinationPath, *repo.Name))
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

func getContents(ctx context.Context, client *github.Client, owner, repo, path, basePath string) {
	fmt.Println("\n\n")

	fileContent, directoryContent, resp, err := client.Repositories.GetContents(ctx, owner, repo, path, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%#v\n", fileContent)
	fmt.Printf("%#v\n", directoryContent)
	fmt.Printf("%#v\n", resp)

	for _, c := range directoryContent {
		fmt.Println(*c.Type, *c.Path, *c.Size, *c.SHA)

		local := filepath.Join(basePath, *c.Path)
		fmt.Println("local:", local)

		switch *c.Type {
		case "file":
			_, err := os.Stat(local)
			if err == nil {
				b, err1 := os.ReadFile(local)
				if err1 == nil {
					sha := calculateGitSHA1(b)
					if *c.SHA == hex.EncodeToString(sha) {
						fmt.Println("no need to update this file, the SHA is the same")
						continue
					}
				}
			}
			downloadContents(ctx, client, c, owner, repo, local)
		case "dir":
			getContents(ctx, client, owner, repo, filepath.Join(path, *c.Path), basePath)
		}
	}
}

func downloadContents(ctx context.Context, client *github.Client, content *github.RepositoryContent, owner, repo, localPath string) {
	if content.Content != nil {
		fmt.Println("content:", *content.Content)
	}

	rc, _, err := client.Repositories.DownloadContents(ctx, owner, repo, *content.Path, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer rc.Close()

	b, err := ioutil.ReadAll(rc)
	if err != nil {
		fmt.Println(err)
		return
	}

	err = os.MkdirAll(filepath.Dir(localPath), 0666)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("Writing the file:", localPath)
	f, err := os.Create(localPath)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer f.Close()
	n, err := f.Write(b)
	if err != nil {
		fmt.Println(err)
	}
	if n != *content.Size {
		fmt.Printf("number of bytes differ, %d vs %d\n", n, *content.Size)
	}
}

// calculateGitSHA1 computes the github sha1 from a slice of bytes.
// The bytes are prepended with: "blob " + filesize + "\0" before runing through sha1.
func calculateGitSHA1(contents []byte) []byte {
	contentLen := len(contents)
	blobSlice := []byte("blob " + strconv.Itoa(contentLen))
	blobSlice = append(blobSlice, '\x00')
	blobSlice = append(blobSlice, contents...)
	h := sha1.New()
	h.Write(blobSlice)
	bs := h.Sum(nil)
	return bs
}
