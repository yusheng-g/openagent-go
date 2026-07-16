// Package keyring wraps the system keychain via go-keyring.
//
// Supported: Linux Secret Service (D-Bus), macOS Keychain, Windows Credential Manager.
package keyring

import (
	"errors"

	gkr "github.com/zalando/go-keyring"
)

// Store wraps the system keychain. Methods accept a service name
// so callers (including WASM plugins) can access keys under different
// namespaces.
type Store struct{}

// Open returns a keyring store. Probes the backend to verify it's available.
func Open() (*Store, error) {
	_, err := gkr.Get("openagent", "__probe__")
	if err != nil && !errors.Is(err, gkr.ErrNotFound) {
		return nil, err
	}
	return &Store{}, nil
}

func (s *Store) Get(service, key string) (string, error) {
	v, err := gkr.Get(service, key)
	if err != nil {
		if errors.Is(err, gkr.ErrNotFound) { return "", nil }
		return "", err
	}
	return v, nil
}

func (s *Store) Set(service, key, value string) error {
	if value == "" { return s.Delete(service, key) }
	return gkr.Set(service, key, value)
}

func (s *Store) Delete(service, key string) error {
	err := gkr.Delete(service, key)
	if errors.Is(err, gkr.ErrNotFound) { return nil }
	return err
}
