// credential.go - Token Storage (Credential Manager + Encrypted Fallback)
//
// Primary storage: Windows Credential Manager (advapi32.dll)
// Fallback: AES-GCM encrypted token in config.json
//
// The fallback exists because tools like CCleaner can wipe the
// Windows Credential Manager, forcing users to re-enter their token.
// With the encrypted backup in config.json, the token is automatically
// restored.
package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"syscall"
	"unsafe"
)

var (
	advapi32       = syscall.NewLazyDLL("advapi32.dll")
	procCredWriteW  = advapi32.NewProc("CredWriteW")
	procCredReadW   = advapi32.NewProc("CredReadW")
	procCredDeleteW = advapi32.NewProc("CredDeleteW")
	procCredFree    = advapi32.NewProc("CredFree")
)

const (
	CRED_TYPE_GENERIC          = 1
	CRED_PERSIST_LOCAL_MACHINE = 2
)

// CREDENTIAL represents a Windows Credential Manager entry (CREDENTIALW struct).
type CREDENTIAL struct {
	Flags              uint32
	Type               uint32
	TargetName         *uint16
	Comment            *uint16
	LastWritten        uint64
	CredentialBlobSize uint32
	CredentialBlob     *byte
	Persist            uint32
	AttributeCount     uint32
	Attributes         uintptr
	TargetAlias        *uint16
	UserName           *uint16
}

// credentialTarget is set per build: release uses production key, debug uses a
// separate key so tokens don't overwrite each other.
var credentialTarget = "TarkovScreenshoter_APIToken"

// Static AES-256 key for config.json token encryption.
// This is obfuscation, not security — prevents casual reading of the config file.
var tokenEncKey = []byte{
	0x4b, 0x69, 0x6c, 0x6c, 0x43, 0x6f, 0x75, 0x6e,
	0x74, 0x65, 0x72, 0x54, 0x61, 0x72, 0x6b, 0x6f,
	0x76, 0x53, 0x74, 0x61, 0x6d, 0x6d, 0x74, 0x69,
	0x73, 0x63, 0x68, 0x32, 0x30, 0x32, 0x36, 0x21,
}

// SaveToken stores the API token in Credential Manager and saves an
// encrypted backup to config.json.
func SaveToken(token string) error {
	// Primary: Credential Manager
	err := saveTokenCredMgr(token)

	// Backup: encrypted in config.json
	cfg, cfgErr := LoadConfig()
	if cfgErr == nil {
		if encrypted, encErr := encryptToken(token); encErr == nil {
			cfg.EncryptedToken = encrypted
			SaveConfig(cfg)
		}
	}

	return err
}

// LoadToken retrieves the API token. Tries Credential Manager first,
// falls back to encrypted config.json backup.
func LoadToken() (string, error) {
	// Try Credential Manager first
	token, err := loadTokenCredMgr()
	if err == nil && token != "" {
		return token, nil
	}

	// Fallback: try encrypted backup from config.json
	cfg, cfgErr := LoadConfig()
	if cfgErr == nil && cfg.EncryptedToken != "" {
		if decrypted, decErr := decryptToken(cfg.EncryptedToken); decErr == nil && decrypted != "" {
			// Restore to Credential Manager for next time
			saveTokenCredMgr(decrypted)
			return decrypted, nil
		}
	}

	return "", err
}

// DeleteToken removes the API token from both Credential Manager and config.json.
func DeleteToken() error {
	// Remove from config.json
	cfg, cfgErr := LoadConfig()
	if cfgErr == nil && cfg.EncryptedToken != "" {
		cfg.EncryptedToken = ""
		SaveConfig(cfg)
	}

	// Remove from Credential Manager
	return deleteTokenCredMgr()
}

// HasToken returns true if a non-empty API token exists.
func HasToken() bool {
	token, err := LoadToken()
	return err == nil && token != ""
}

// --- Credential Manager functions ---

func saveTokenCredMgr(token string) error {
	targetName, _ := syscall.UTF16PtrFromString(credentialTarget)
	userName, _ := syscall.UTF16PtrFromString("api_token")

	tokenBytes := []byte(token)

	cred := CREDENTIAL{
		Type:               CRED_TYPE_GENERIC,
		TargetName:         targetName,
		CredentialBlobSize: uint32(len(tokenBytes)),
		CredentialBlob:     &tokenBytes[0],
		Persist:            CRED_PERSIST_LOCAL_MACHINE,
		UserName:           userName,
	}

	ret, _, err := procCredWriteW.Call(uintptr(unsafe.Pointer(&cred)), 0)
	if ret == 0 {
		return err
	}
	return nil
}

func loadTokenCredMgr() (string, error) {
	targetName, _ := syscall.UTF16PtrFromString(credentialTarget)

	var cred *CREDENTIAL
	ret, _, err := procCredReadW.Call(
		uintptr(unsafe.Pointer(targetName)),
		CRED_TYPE_GENERIC,
		0,
		uintptr(unsafe.Pointer(&cred)),
	)

	if ret == 0 {
		return "", err
	}
	defer procCredFree.Call(uintptr(unsafe.Pointer(cred)))

	token := make([]byte, cred.CredentialBlobSize)
	for i := uint32(0); i < cred.CredentialBlobSize; i++ {
		token[i] = *(*byte)(unsafe.Pointer(uintptr(unsafe.Pointer(cred.CredentialBlob)) + uintptr(i)))
	}

	return string(token), nil
}

func deleteTokenCredMgr() error {
	targetName, _ := syscall.UTF16PtrFromString(credentialTarget)

	ret, _, err := procCredDeleteW.Call(
		uintptr(unsafe.Pointer(targetName)),
		CRED_TYPE_GENERIC,
		0,
	)

	if ret == 0 {
		return err
	}
	return nil
}

// --- AES-GCM encryption ---

func encryptToken(plaintext string) (string, error) {
	block, err := aes.NewCipher(tokenEncKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func decryptToken(encoded string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(tokenEncKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
