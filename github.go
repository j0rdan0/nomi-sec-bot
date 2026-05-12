package main

import (
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

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}

func githubAPIRequest(url string) ([]byte, error) {
	log.Printf("Requesting: %s", url)
	req, err := http.NewRequest("GET", url, nil)
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

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github api error: %s (url: %s)", resp.Status, url)
	}

	return io.ReadAll(resp.Body)
}

func GetRecentCommits() ([]Commit, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/commits", owner, repo)
	data, err := githubAPIRequest(url)
	if err != nil {
		return nil, err
	}

	var commits []Commit
	err = json.Unmarshal(data, &commits)
	return commits, err
}

func GetCommitChangedFiles(sha string) ([]string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/commits/%s", owner, repo, sha)
	data, err := githubAPIRequest(url)
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
		if strings.HasSuffix(f.Filename, ".json") && !strings.Contains(f.Filename, "/") {
			// This filters for JSON files in the root, but PoCs are in year folders.
			// The research said they are in folders like /2024/CVE-*.json
		}
		if strings.HasSuffix(f.Filename, ".json") {
			files = append(files, f.Filename)
		}
	}
	return files, nil
}

func FetchPoCInfo(filePath string) ([]PoCInfo, error) {
	url := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/master/%s", owner, repo, filePath)
	data, err := githubAPIRequest(url)
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

func GetCVEsForYear(year string, count int) ([]string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s", owner, repo, year)
	data, err := githubAPIRequest(url)
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
