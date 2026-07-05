//go:build !windows

package syncconfig

import "errors"

// dpapiAvailable reports whether at-rest key protection is supported on this
// platform. Non-Windows platforms rely on 0600 file permissions instead.
const dpapiAvailable = false

var errDPAPIUnsupported = errors.New("dpapi key protection is only supported on windows")

func protectAPIKey(string) (string, error)   { return "", errDPAPIUnsupported }
func unprotectAPIKey(string) (string, error) { return "", errDPAPIUnsupported }
