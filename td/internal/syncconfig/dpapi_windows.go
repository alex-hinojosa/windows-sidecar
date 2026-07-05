//go:build windows

package syncconfig

import (
	"encoding/base64"
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

// dpapiAvailable reports whether at-rest key protection is supported on this
// platform. On Windows, auth material is wrapped with DPAPI because POSIX
// file modes (0600) are a no-op there.
const dpapiAvailable = true

// protectAPIKey encrypts key with the Windows Data Protection API scoped to
// the current user and returns the ciphertext as base64.
func protectAPIKey(key string) (string, error) {
	data := []byte(key)
	if len(data) == 0 {
		return "", fmt.Errorf("dpapi protect: empty key")
	}
	in := windows.DataBlob{Size: uint32(len(data)), Data: &data[0]}
	var out windows.DataBlob
	if err := windows.CryptProtectData(&in, nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out); err != nil {
		return "", fmt.Errorf("dpapi protect: %w", err)
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))
	return base64.StdEncoding.EncodeToString(unsafe.Slice(out.Data, out.Size)), nil
}

// unprotectAPIKey decrypts a base64 DPAPI ciphertext produced by
// protectAPIKey. Fails when the ciphertext was created by a different user
// or on a different machine.
func unprotectAPIKey(b64 string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return "", fmt.Errorf("dpapi unprotect: decode base64: %w", err)
	}
	if len(data) == 0 {
		return "", fmt.Errorf("dpapi unprotect: empty ciphertext")
	}
	in := windows.DataBlob{Size: uint32(len(data)), Data: &data[0]}
	var out windows.DataBlob
	if err := windows.CryptUnprotectData(&in, nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out); err != nil {
		return "", fmt.Errorf("dpapi unprotect: %w", err)
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))
	return string(unsafe.Slice(out.Data, out.Size)), nil
}
