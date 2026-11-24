// authenticode.go - Publisher Check for Self-Update
//
// Before a downloaded update replaces the running exe, its Authenticode
// signature is checked: the file must carry a valid signature that chains to
// a root Windows trusts, and the signing certificate must name the expected
// publisher. A release asset that was swapped or tampered with — for example
// through a compromised GitHub account — fails this check, because it cannot
// be signed with the project's certificate.
//
// The check pins the certificate's common name, not its thumbprint, so that
// renewing the certificate does not lock existing installs out of updates.
package main

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// expectedPublisher is the common name (CN) of the code signing certificate
// used for official releases. See build.sh / build.bat.
const expectedPublisher = "Open Source Developer Martin Wilke"

var (
	crypt32              = syscall.NewLazyDLL("crypt32.dll")
	procCryptMsgGetParam = crypt32.NewProc("CryptMsgGetParam")
	procCryptMsgClose    = crypt32.NewProc("CryptMsgClose")
)

const cmsgSignerCertInfoParam = 7 // CMSG_SIGNER_CERT_INFO_PARAM

// oidCommonName is szOID_COMMON_NAME, NUL-terminated for the Win32 API.
var oidCommonName = []byte("2.5.4.3\x00")

// verifyPublisher returns nil only if the file at path has a valid, trusted
// Authenticode signature made by expectedPublisher.
func verifyPublisher(path string) error {
	if err := verifyTrust(path); err != nil {
		return fmt.Errorf("signature not valid: %w", err)
	}
	signer, err := signerCommonName(path)
	if err != nil {
		return fmt.Errorf("cannot read signer: %w", err)
	}
	if signer != expectedPublisher {
		return fmt.Errorf("signed by %q, expected %q", signer, expectedPublisher)
	}
	return nil
}

// verifyTrust runs WinVerifyTrust: the file hash must match the signature and
// the certificate chain must end in a trusted root. No UI is shown.
func verifyTrust(path string) error {
	path16, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}

	fileInfo := windows.WinTrustFileInfo{
		Size:     uint32(unsafe.Sizeof(windows.WinTrustFileInfo{})),
		FilePath: path16,
	}
	data := windows.WinTrustData{
		Size:                            uint32(unsafe.Sizeof(windows.WinTrustData{})),
		UIChoice:                        windows.WTD_UI_NONE,
		RevocationChecks:                windows.WTD_REVOKE_WHOLECHAIN,
		UnionChoice:                     windows.WTD_CHOICE_FILE,
		FileOrCatalogOrBlobOrSgnrOrCert: unsafe.Pointer(&fileInfo),
		StateAction:                     windows.WTD_STATEACTION_VERIFY,
		ProvFlags:                       windows.WTD_REVOCATION_CHECK_CHAIN_EXCLUDE_ROOT,
	}

	verifyErr := windows.WinVerifyTrustEx(windows.InvalidHWND, &windows.WINTRUST_ACTION_GENERIC_VERIFY_V2, &data)

	// Release the state WinVerifyTrust allocated, whatever the verdict was.
	data.StateAction = windows.WTD_STATEACTION_CLOSE
	windows.WinVerifyTrustEx(windows.InvalidHWND, &windows.WINTRUST_ACTION_GENERIC_VERIFY_V2, &data)

	return verifyErr
}

// signerCommonName returns the CN of the certificate that made the file's
// embedded (primary) Authenticode signature.
func signerCommonName(path string) (string, error) {
	path16, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return "", err
	}

	var store, msg windows.Handle
	err = windows.CryptQueryObject(
		windows.CERT_QUERY_OBJECT_FILE,
		unsafe.Pointer(path16),
		windows.CERT_QUERY_CONTENT_FLAG_PKCS7_SIGNED_EMBED,
		windows.CERT_QUERY_FORMAT_FLAG_BINARY,
		0, nil, nil, nil,
		&store, &msg, nil,
	)
	if err != nil {
		return "", err
	}
	defer windows.CertCloseStore(store, 0)
	defer procCryptMsgClose.Call(uintptr(msg))

	// The signer is identified inside the signature by issuer + serial number.
	var size uint32
	ret, _, callErr := procCryptMsgGetParam.Call(uintptr(msg), cmsgSignerCertInfoParam, 0, 0, uintptr(unsafe.Pointer(&size)))
	if ret == 0 || size == 0 {
		return "", fmt.Errorf("CryptMsgGetParam (size): %v", callErr)
	}
	buf := make([]byte, size)
	ret, _, callErr = procCryptMsgGetParam.Call(uintptr(msg), cmsgSignerCertInfoParam, 0,
		uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)))
	if ret == 0 {
		return "", fmt.Errorf("CryptMsgGetParam: %v", callErr)
	}

	// Look that certificate up among the ones shipped in the signature.
	cert, err := windows.CertFindCertificateInStore(
		store,
		windows.X509_ASN_ENCODING|windows.PKCS_7_ASN_ENCODING,
		0,
		windows.CERT_FIND_SUBJECT_CERT,
		unsafe.Pointer(&buf[0]),
		nil,
	)
	if err != nil {
		return "", err
	}
	defer windows.CertFreeCertificateContext(cert)

	var name [256]uint16
	chars := windows.CertGetNameString(cert, windows.CERT_NAME_ATTR_TYPE, 0,
		unsafe.Pointer(&oidCommonName[0]), &name[0], uint32(len(name)))
	if chars <= 1 {
		return "", fmt.Errorf("signing certificate has no common name")
	}
	return windows.UTF16ToString(name[:]), nil
}
