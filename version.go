// version.go - Automatic Update Checker
//
// This file handles version checking and update notifications:
// - Queries GitHub Releases API for the latest version
// - Compares current version with latest release
// - On startup: shows dialog asking to download
// - While running: blinks tray icon and adds menu entry
// - Checks on startup and every 30 minutes
//
// The update check is lightweight (~1-2KB JSON response) and
// uses minimal CPU (~5-10ms per check).
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/lxn/walk"
)

const (
	CurrentVersion = "1.0.0-rc1"
	GithubRepo     = "miwidot/tarkov-killscreen"
)

// GithubRelease represents the JSON response from the GitHub Releases API.
type GithubRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
	Name    string `json:"name"`
}

var (
	updateAvailable bool
	latestRelease   GithubRelease
	updateAction    *walk.Action // Tray menu update entry
	blinkStop       chan struct{}
)

// StartUpdateChecker runs update check on startup (with dialog) and every
// 30 minutes (tray icon blink only).
func StartUpdateChecker() {
	// Check on startup — show dialog if update found
	checkForUpdates(true)

	// Then check every 30 minutes — blink only, no dialog
	ticker := time.NewTicker(30 * time.Minute)
	go func() {
		for range ticker.C {
			debugLn("[UPDATE] Periodic check...")
			checkForUpdates(false)
		}
	}()
}

// checkForUpdates queries GitHub for a newer release (including pre-releases).
// If showDialog is true, a message box is shown. Otherwise the tray icon blinks.
func checkForUpdates(showDialog bool) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases?per_page=5", GithubRepo)

	resp, err := http.Get(url)
	if err != nil {
		debugLn("[UPDATE] Failed to check for updates:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		debugLn("[UPDATE] GitHub API returned:", resp.StatusCode)
		return
	}

	var releases []GithubRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		debugLn("[UPDATE] Failed to parse response:", err)
		return
	}

	if len(releases) == 0 {
		return
	}

	// Find the newest release by version comparison
	var newest *GithubRelease
	for i := range releases {
		if newest == nil || isNewerVersion(releases[i].TagName, newest.TagName) {
			newest = &releases[i]
		}
	}

	debugLog("[UPDATE] Current: %s, Latest: %s\n", CurrentVersion, newest.TagName)

	if isNewerVersion(newest.TagName, CurrentVersion) {
		latestRelease = *newest
		updateAvailable = true

		if showDialog {
			showUpdateDialog(*newest)
		} else {
			showUpdateTray(*newest)
		}
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

	// Parse semver parts for proper comparison
	latestParts := parseVersion(latest)
	currentParts := parseVersion(current)

	// Compare major.minor.patch first
	for i := 0; i < 3; i++ {
		if latestParts[i] != currentParts[i] {
			return latestParts[i] > currentParts[i]
		}
	}

	// Same major.minor.patch — compare pre-release (beta number)
	// No pre-release (stable) > any pre-release (beta)
	if latestParts[3] == 0 && currentParts[3] > 0 {
		return true // stable > beta
	}
	if latestParts[3] > 0 && currentParts[3] == 0 {
		return false // beta < stable
	}
	return latestParts[3] > currentParts[3]
}

// parseVersion parses "1.0.0-beta12" into [1, 0, 0, 12].
// Returns [major, minor, patch, preRelease]. Stable releases have preRelease=0.
func parseVersion(version string) [4]int {
	var parts [4]int

	// Split off pre-release suffix (e.g. "-beta12")
	base := version
	preRelease := ""
	if idx := strings.IndexByte(version, '-'); idx >= 0 {
		base = version[:idx]
		preRelease = version[idx+1:]
	}

	// Parse major.minor.patch
	segments := strings.Split(base, ".")
	for i := 0; i < 3 && i < len(segments); i++ {
		fmt.Sscanf(segments[i], "%d", &parts[i])
	}

	// Parse pre-release number (e.g. "beta12" → 12)
	if preRelease != "" {
		for _, c := range preRelease {
			if c >= '0' && c <= '9' {
				parts[3] = parts[3]*10 + int(c-'0')
			}
		}
		if parts[3] == 0 {
			parts[3] = 1 // "beta" without number = 1
		}
	}

	return parts
}

// showUpdateDialog shows a Windows message box with update info (startup only).
func showUpdateDialog(release GithubRelease) {
	message := fmt.Sprintf(T("update.message"), CurrentVersion, release.TagName)

	result := walk.MsgBox(
		nil,
		T("update.title"),
		message,
		walk.MsgBoxIconInformation|walk.MsgBoxYesNo,
	)

	if result == walk.DlgCmdYes {
		openBrowser(release.HTMLURL)
	}
}

// showUpdateTray adds an update entry to the tray menu and blinks the icon.
func showUpdateTray(release GithubRelease) {
	if notifyIcon == nil {
		return
	}

	// Add update menu entry if not already present
	if updateAction == nil {
		updateAction = walk.NewAction()
		updateAction.Triggered().Attach(func() {
			openBrowser(latestRelease.HTMLURL)
		})
		// Insert at top of menu
		notifyIcon.ContextMenu().Actions().Insert(0, updateAction)
		sep := walk.NewSeparatorAction()
		notifyIcon.ContextMenu().Actions().Insert(1, sep)
	}
	updateAction.SetText(fmt.Sprintf(T("update.available"), release.TagName))

	// Show balloon notification
	showBalloon(T("update.title"), fmt.Sprintf(T("update.available"), release.TagName))

	// Blink tray icon
	startIconBlink()
}

// startIconBlink alternates the tray icon between normal and an "update" icon.
func startIconBlink() {
	if blinkStop != nil {
		return // Already blinking
	}
	blinkStop = make(chan struct{})

	normalIcon, _ := walk.NewIconFromImageForDPI(createIconImage(), 96)
	updateIcon, _ := walk.NewIconFromImageForDPI(createUpdateIconImage(), 96)

	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		show := false
		for {
			select {
			case <-blinkStop:
				if notifyIcon != nil && normalIcon != nil {
					notifyIcon.SetIcon(normalIcon)
				}
				return
			case <-ticker.C:
				if notifyIcon == nil {
					return
				}
				show = !show
				if show && updateIcon != nil {
					notifyIcon.SetIcon(updateIcon)
				} else if normalIcon != nil {
					notifyIcon.SetIcon(normalIcon)
				}
			}
		}
	}()
}

// openBrowser opens the default browser with the given URL
func openBrowser(url string) {
	exec.Command("cmd", "/c", "start", url).Start()
}
