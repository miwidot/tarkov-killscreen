// selfupdate.go - In-Place Self-Update
//
// Downloads the new exe from GitHub Releases, replaces the current
// exe using the rename trick (Windows allows renaming a running exe),
// and restarts the app. Cleans up the old exe on next start.
//
// Flow:
//   1. Download new exe to <current>.new
//   2. Verify signature (signtool) if available
//   3. Rename running exe to <current>.old
//   4. Rename .new to <current>
//   5. Start new exe, exit current
//   6. On next start: delete .old
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

// GithubAsset represents a release asset from the GitHub API.
type GithubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// GithubReleaseWithAssets extends GithubRelease with asset info.
type GithubReleaseWithAssets struct {
	TagName string        `json:"tag_name"`
	HTMLURL string        `json:"html_url"`
	Name    string        `json:"name"`
	Assets  []GithubAsset `json:"assets"`
}

// selfUpdateTestPath is set via --selfupdate-test flag for local testing.
// When set, the updater copies from this path instead of downloading.
var selfUpdateTestPath string

// CleanupOldExe removes leftover .old files from a previous update.
// Called on every app start.
func CleanupOldExe() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	oldPath := exe + ".old"
	if _, err := os.Stat(oldPath); err == nil {
		if err := os.Remove(oldPath); err != nil {
			debugLn("[UPDATE] Failed to clean up old exe:", err)
		} else {
			debugLn("[UPDATE] Cleaned up old exe")
		}
	}
}

// PerformSelfUpdate downloads the new release and replaces the running exe.
// Returns nil on success (caller should exit so the new exe can start).
func PerformSelfUpdate(release GithubRelease) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot determine exe path: %w", err)
	}

	newPath := exe + ".new"
	oldPath := exe + ".old"

	// Step 1: Get the new exe
	if selfUpdateTestPath != "" {
		// Test mode: copy from local file
		debugLn("[UPDATE] Test mode: copying from", selfUpdateTestPath)
		if err := copyFile(selfUpdateTestPath, newPath); err != nil {
			return fmt.Errorf("copy failed: %w", err)
		}
	} else {
		// Production: download from GitHub
		downloadURL, err := findExeAssetURL(release)
		if err != nil {
			return err
		}

		debugLn("[UPDATE] Downloading from", downloadURL)
		if err := downloadFile(downloadURL, newPath); err != nil {
			os.Remove(newPath)
			return fmt.Errorf("download failed: %w", err)
		}
	}

	// Step 2: Verify the downloaded exe is actually an exe (basic check)
	if err := verifyExe(newPath); err != nil {
		os.Remove(newPath)
		return fmt.Errorf("verification failed: %w", err)
	}

	// Step 3: Rename current exe to .old (Windows allows renaming a running exe)
	// Remove stale .old first if cleanup failed last time
	os.Remove(oldPath)
	if err := os.Rename(exe, oldPath); err != nil {
		os.Remove(newPath)
		return fmt.Errorf("cannot rename current exe: %w", err)
	}

	// Step 4: Rename .new to current
	if err := os.Rename(newPath, exe); err != nil {
		// Rollback: try to restore old exe
		os.Rename(oldPath, exe)
		return fmt.Errorf("cannot rename new exe: %w", err)
	}

	debugLn("[UPDATE] Exe replaced successfully")

	// Step 5: Start the new exe and exit
	fmt.Println("[UPDATE] Restarting...")
	startNewExe(exe)
	return nil
}

// findExeAssetURL queries the GitHub release for the .exe download URL.
func findExeAssetURL(release GithubRelease) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/tags/%s", GithubRepo, release.TagName)

	resp, err := apiClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("cannot fetch release details: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var fullRelease GithubReleaseWithAssets
	if err := json.NewDecoder(resp.Body).Decode(&fullRelease); err != nil {
		return "", fmt.Errorf("cannot parse release: %w", err)
	}

	// Find the screenshoter exe asset by exact name match.
	// We deliberately don't accept any *.exe in the release because releases
	// can ship additional tools (e.g. diagnose.exe) that must not be picked
	// up as the main app's update.
	for _, asset := range fullRelease.Assets {
		if strings.EqualFold(asset.Name, "screenshoter.exe") {
			return asset.BrowserDownloadURL, nil
		}
	}

	return "", fmt.Errorf("no screenshoter.exe asset found in release %s", release.TagName)
}

// downloadFile downloads a URL to a local file with progress logging.
func downloadFile(url, destPath string) error {
	resp, err := apiClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	written, err := io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	debugLog("[UPDATE] Downloaded %d bytes\n", written)
	return nil
}

// copyFile copies src to dst (for test mode).
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

// verifyExe does a basic check that the file is a valid PE executable.
func verifyExe(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	// Check MZ header
	header := make([]byte, 2)
	if _, err := f.Read(header); err != nil {
		return fmt.Errorf("cannot read header: %w", err)
	}
	if header[0] != 'M' || header[1] != 'Z' {
		return fmt.Errorf("not a valid exe (missing MZ header)")
	}

	// Check file size is reasonable (> 1MB, < 100MB)
	info, err := f.Stat()
	if err != nil {
		return err
	}
	if info.Size() < 1*1024*1024 {
		return fmt.Errorf("exe too small (%d bytes)", info.Size())
	}
	if info.Size() > 100*1024*1024 {
		return fmt.Errorf("exe too large (%d bytes)", info.Size())
	}

	return nil
}

// startNewExe launches the new exe and detaches it from the current process.
func startNewExe(exePath string) {
	verb, _ := syscall.UTF16PtrFromString("open")
	exe, _ := syscall.UTF16PtrFromString(exePath)
	dir, _ := syscall.UTF16PtrFromString(filepath.Dir(exePath))

	procShellExecuteW.Call(
		0,
		uintptr(unsafe.Pointer(verb)),
		uintptr(unsafe.Pointer(exe)),
		0,
		uintptr(unsafe.Pointer(dir)),
		1, // SW_SHOWNORMAL
	)
}
