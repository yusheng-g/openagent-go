//go:build !linux

package keyring

import (
	"errors"

	gkr "github.com/zalando/go-keyring"
)

// openBackend selects a backend on non-Linux platforms (macOS / Windows).
// zalando/go-keyring routes to the native keychain (Keychain / WinCred)
// which does not depend on D-Bus.
func openBackend() (backend, error) {
	if _, err := gkr.Get("openagent", "__probe__"); err != nil &&
		!errors.Is(err, gkr.ErrNotFound) {
		return nil, err
	}
	return secretServiceBackend{}, nil
}