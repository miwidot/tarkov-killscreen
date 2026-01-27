package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"

	"github.com/lxn/walk"
)

const (
	CurrentVersion = "alpha3"
	GithubRepo     = "miwidot/tarkov-killscreen"
)

type GithubRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
	Name    string `json:"name"`
}

// CheckForUpdates checks GitHub for a newer release
func CheckForUpdates() {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", GithubRepo)

	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("[UPDATE] Failed to check for updates:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Println("[UPDATE] GitHub API returned:", resp.StatusCode)
		return
	}

	var release GithubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		fmt.Println("[UPDATE] Failed to parse response:", err)
		return
	}

	fmt.Printf("[UPDATE] Current: %s, Latest: %s\n", CurrentVersion, release.TagName)

	if isNewerVersion(release.TagName, CurrentVersion) {
		showUpdateDialog(release)
	}
}

// isNewerVersion compares version strings (simple string comparison for alpha/beta tags)
func isNewerVersion(latest, current string) bool {
	// Normalize versions
	latest = strings.ToLower(strings.TrimPrefix(latest, "v"))
	current = strings.ToLower(strings.TrimPrefix(current, "v"))

	// Same version
	if latest == current {
		return false
	}

	// For alpha/beta versions, compare numerically if possible
	// alpha2 vs alpha3 -> extract numbers
	latestNum := extractVersionNumber(latest)
	currentNum := extractVersionNumber(current)

	if latestNum > 0 && currentNum > 0 {
		return latestNum > currentNum
	}

	// Fallback: simple string comparison
	return latest > current
}

// extractVersionNumber extracts the number from version strings like "alpha2", "beta3", "v1.2.3"
func extractVersionNumber(version string) int {
	var num int
	// Try to find a number in the string
	for _, c := range version {
		if c >= '0' && c <= '9' {
			num = num*10 + int(c-'0')
		}
	}
	return num
}

// showUpdateDialog shows a Windows message box with update info
func showUpdateDialog(release GithubRelease) {
	message := fmt.Sprintf(
		"Eine neue Version ist verfügbar!\n\n"+
			"Aktuelle Version: %s\n"+
			"Neue Version: %s\n\n"+
			"Jetzt herunterladen?",
		CurrentVersion,
		release.TagName,
	)

	result := walk.MsgBox(
		nil,
		"Update verfügbar",
		message,
		walk.MsgBoxIconInformation|walk.MsgBoxYesNo,
	)

	if result == walk.DlgCmdYes {
		// Open browser to release page
		openBrowser(release.HTMLURL)
	}
}

// openBrowser opens the default browser with the given URL
func openBrowser(url string) {
	exec.Command("cmd", "/c", "start", url).Start()
}
