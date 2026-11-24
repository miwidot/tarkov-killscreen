package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// signedReleaseExe returns the path of a locally built, signed release exe,
// or skips the test when there is none (fresh checkout, CI).
func signedReleaseExe(t *testing.T) string {
	t.Helper()
	const path = "screenshoter.exe"
	if _, err := os.Stat(path); err != nil {
		t.Skip("no signed screenshoter.exe in the working directory")
	}
	return path
}

func TestVerifyPublisher_AcceptsOwnRelease(t *testing.T) {
	if err := verifyPublisher(signedReleaseExe(t)); err != nil {
		t.Fatal("own signed release rejected:", err)
	}
}

func TestVerifyPublisher_RejectsTamperedExe(t *testing.T) {
	data, err := os.ReadFile(signedReleaseExe(t))
	if err != nil {
		t.Fatal(err)
	}
	// Flip one byte in the middle of the code; the signature stays attached
	// but no longer matches the file.
	data[len(data)/2] ^= 0xFF
	tampered := filepath.Join(t.TempDir(), "tampered.exe")
	if err := os.WriteFile(tampered, data, 0644); err != nil {
		t.Fatal(err)
	}

	if err := verifyPublisher(tampered); err == nil {
		t.Fatal("tampered exe was accepted")
	}
}

func TestVerifyPublisher_RejectsUnsignedExe(t *testing.T) {
	// The test binary itself is a freshly built, unsigned exe.
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	if err := verifyPublisher(self); err == nil {
		t.Fatal("unsigned exe was accepted")
	}
}

func TestVerifyPublisher_RejectsOtherPublisher(t *testing.T) {
	// A valid signature alone must not be enough: find any installed tool
	// that is validly signed by someone else.
	var other string
	for _, name := range []string{"git.exe", "pwsh.exe", "gh.exe", "code.cmd", "node.exe"} {
		path, err := exec.LookPath(name)
		if err == nil && verifyTrust(path) == nil {
			other = path
			break
		}
	}
	if other == "" {
		t.Skip("no validly signed third-party exe found on PATH")
	}
	t.Log("using", other)

	err := verifyPublisher(other)
	if err == nil {
		t.Fatal("exe from another publisher was accepted")
	}
	if !strings.Contains(err.Error(), "expected") {
		t.Fatalf("rejected for the wrong reason: %v", err)
	}
}
