//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const (
	currentVersion  = "1.0.2"
	releasesAPIURL  = "https://api.github.com/repos/buildcraftlabs/hotfix/releases/latest"
	releasesPageURL = "https://github.com/buildcraftlabs/hotfix/releases/latest"
)

type githubRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

// checkForUpdates fetches the latest GitHub release and opens the browser
// if a newer version is available. It is safe to call from any goroutine.
func checkForUpdates() {
	logf("updater: checking for updates (current: %s)", currentVersion)

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodGet, releasesAPIURL, nil)
	if err != nil {
		logf("updater: build request error: %v", err)
		return
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", fmt.Sprintf("Hotfix/%s", currentVersion))

	resp, err := client.Do(req)
	if err != nil {
		logf("updater: request error: %v", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logf("updater: read error: %v", err)
		return
	}

	var release githubRelease
	if err := json.Unmarshal(body, &release); err != nil {
		logf("updater: JSON parse error: %v", err)
		return
	}

	tag := release.TagName
	if strings.HasPrefix(tag, "v") {
		tag = tag[1:]
	}

	logf("updater: latest release is %s", tag)

	if isNewer(tag, currentVersion) {
		logf("updater: update available (%s → %s), opening browser", currentVersion, tag)
		url := release.HTMLURL
		if url == "" {
			url = releasesPageURL
		}
		openURL(url)
	} else {
		logf("updater: already up to date")
	}
}

// isNewer returns true when remote semver is greater than installed semver.
func isNewer(remote, installed string) bool {
	rv := parseSemver(remote)
	cv := parseSemver(installed)
	for i := 0; i < 3; i++ {
		if rv[i] > cv[i] {
			return true
		}
		if rv[i] < cv[i] {
			return false
		}
	}
	return false
}

func parseSemver(v string) [3]int {
	parts := strings.SplitN(v, ".", 3)
	var result [3]int
	for i, p := range parts {
		if i >= 3 {
			break
		}
		n, _ := strconv.Atoi(strings.TrimSpace(p))
		result[i] = n
	}
	return result
}

// openURL opens a URL in the default Windows browser.
func openURL(url string) {
	cmd := exec.Command("cmd", "/c", "start", url)
	if err := cmd.Start(); err != nil {
		logf("updater: open URL error: %v", err)
	}
}
