// credential.go - Windows Credential Manager Integration
//
// This file provides secure storage for the API token using
// Windows Credential Manager (advapi32.dll).
//
// The API token is stored encrypted by Windows, not in plain text.
// This is the same secure storage used by Windows for network
// passwords, browser credentials, and other sensitive data.
//
// Functions:
// - SaveToken: Store API token in Credential Manager
// - LoadToken: Retrieve API token from Credential Manager
// - HasToken: Check if a token exists
// - DeleteToken: Remove token from Credential Manager
package main

import (
	"syscall"
	"unsafe"
)

var (
	advapi32            = syscall.NewLazyDLL("advapi32.dll")
	procCredWriteW      = advapi32.NewProc("CredWriteW")
	procCredReadW       = advapi32.NewProc("CredReadW")
	procCredDeleteW     = advapi32.NewProc("CredDeleteW")
	procCredFree        = advapi32.NewProc("CredFree")
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

const credentialTarget = "TarkovScreenshoter_APIToken"

// SaveToken stores the API token in Windows Credential Manager.
func SaveToken(token string) error {
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

// LoadToken retrieves the API token from Windows Credential Manager.
func LoadToken() (string, error) {
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

// DeleteToken removes the API token from Windows Credential Manager.
func DeleteToken() error {
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

// HasToken returns true if a non-empty API token exists in Credential Manager.
func HasToken() bool {
	token, err := LoadToken()
	return err == nil && token != ""
}
