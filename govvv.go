package main

//NOTE: All launcher releases are built with the following command:
//  govvv build -ldflags="-s -w"

import (
	"fmt"
	"runtime"
)

var (
	//Set by govvv, used for launcher self-update checks
	GitBranch, GitCommit, GitCommitFull, GitCommitMsg, GitState, BuildDate string

	//Tries to generate a clean BuildID
	BuildID string = fmt.Sprintf("sflaunch-%s_%s-%s_%s-%s", GitState, GitBranch, GitCommit, BuildDate, runtime.Version())
)

func init() {
	//Fix empty build IDs
	if BuildID == "sflaunch-_-_-" + runtime.Version() {
		BuildID = "sflaunch-" + runtime.Version()
	}
}

type GitHubRelease struct {
	ID int64 `json:"id"`
	Author GitHubAuthor `json:"author"`
	Name string `json:"name"`
	TagName string `json:"tag_name"`
	Assets []*GitHubAsset `json:"assets"`
}

type GitHubAuthor struct {
	Login string `json:"login"`
}

type GitHubAsset struct {
	ContentType string `json:"content_type"`
	DownloadCount int `json:"download_count"`
	BrowserDownloadURL string `json:"browser_download_url"`
}
