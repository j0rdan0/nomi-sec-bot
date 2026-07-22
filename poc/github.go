package poc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	owner = "nomi-sec"
	repo  = "PoC-in-GitHub"
)

type Commit struct {
	SHA string `json:"sha"`
}

type CommitDetail struct {
	Files []CommitFile `json:"files"`
}

type CommitFile struct {
	Filename string `json:"filename"`
	Status   string `json:"status"` // added, modified, etc.
}

type PoCInfo struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	RepositoryURL string `json:"html_url"`
	Description   string `json:"description"`
}

var ErrNotFound = fmt.Errorf("resource not found")

var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

func githubAPIRequest(ctx context.Context, url string) ([]byte, error) {
	log.Printf("Requesting: %s", url)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	if !strings.Contains(url, "raw.githubusercontent.com") {
		token := os.Getenv("GITHUB_TOKEN")
		if token != "" {
			req.Header.Set("Authorization", "token "+token)
		}
	}
	req.Header.Set("User-Agent", "nomi-sec-bot")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github api error: %s (url: %s)", resp.Status, url)
	}

	return io.ReadAll(resp.Body)
}

func GetRecentCommits(ctx context.Context) ([]Commit, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/commits", owner, repo)
	data, err := githubAPIRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	var commits []Commit
	err = json.Unmarshal(data, &commits)
	return commits, err
}

type CommitInfoResponse struct {
	Commit struct {
		Committer struct {
			Date string `json:"date"`
		} `json:"committer"`
	} `json:"commit"`
}

func GetCommitDate(ctx context.Context, sha string) (time.Time, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/commits/%s", owner, repo, sha)
	data, err := githubAPIRequest(ctx, url)
	if err != nil {
		return time.Time{}, err
	}

	var res CommitInfoResponse
	err = json.Unmarshal(data, &res)
	if err != nil {
		return time.Time{}, err
	}

	return time.Parse(time.RFC3339, res.Commit.Committer.Date)
}

func GetCommitChangedFiles(ctx context.Context, sha string) ([]string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/commits/%s", owner, repo, sha)
	data, err := githubAPIRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	var detail CommitDetail
	err = json.Unmarshal(data, &detail)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, f := range detail.Files {
		if strings.HasSuffix(f.Filename, ".json") {
			files = append(files, f.Filename)
		}
	}
	return files, nil
}

func FetchPoCInfo(ctx context.Context, filePath string) ([]PoCInfo, error) {
	url := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/master/%s", owner, repo, filePath)
	data, err := githubAPIRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	var infos []PoCInfo
	err = json.Unmarshal(data, &infos)
	if err != nil {
		return nil, err
	}
	return infos, nil
}

func GetCVEsForYear(ctx context.Context, year string, count int) ([]string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s", owner, repo, year)
	data, err := githubAPIRequest(ctx, url)
	if err != nil {
		return nil, err
	}

	var contents []struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}
	err = json.Unmarshal(data, &contents)
	if err != nil {
		return nil, err
	}

	var cves []string
	for _, c := range contents {
		if c.Type == "file" && strings.HasSuffix(c.Name, ".json") {
			cves = append(cves, strings.TrimSuffix(c.Name, ".json"))
			if len(cves) >= count {
				break
			}
		}
	}
	return cves, nil
}
