package main

import (
	"testing"
)

// useTempConfig points the config directory at an empty temp dir and loads a
// fresh global config from it, restoring the previous global afterwards.
func useTempConfig(t *testing.T) {
	t.Helper()
	t.Setenv("APPDATA", t.TempDir())
	saved := config
	t.Cleanup(func() { config = saved })

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	config = cfg
}

// The backup used to be written straight to the file while the in-memory
// config stayed stale, so the next SaveConfig of that struct wiped it.
func TestTokenBackup_SurvivesLaterConfigSave(t *testing.T) {
	useTempConfig(t)

	encrypted, err := encryptToken("token-one")
	if err != nil {
		t.Fatal(err)
	}
	writeTokenBackup(encrypted)

	// Any unrelated save of the live config, e.g. picking an event.
	config.KillEventID = "some-event"
	if err := SaveConfig(config); err != nil {
		t.Fatal(err)
	}

	onDisk, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	got, err := decryptToken(onDisk.EncryptedToken)
	if err != nil {
		t.Fatal("backup missing or unreadable after config save:", err)
	}
	if got != "token-one" {
		t.Fatalf("backup holds %q, want %q", got, "token-one")
	}
}

func TestTokenBackup_FollowsTokenChangeAndDelete(t *testing.T) {
	useTempConfig(t)

	first, _ := encryptToken("old-token")
	writeTokenBackup(first)
	second, _ := encryptToken("new-token")
	writeTokenBackup(second)
	SaveConfig(config)

	onDisk, _ := LoadConfig()
	if got, _ := decryptToken(onDisk.EncryptedToken); got != "new-token" {
		t.Fatalf("backup holds %q after token change, want %q", got, "new-token")
	}

	writeTokenBackup("")
	SaveConfig(config)
	onDisk, _ = LoadConfig()
	if onDisk.EncryptedToken != "" {
		t.Fatal("backup still present after delete")
	}
}
