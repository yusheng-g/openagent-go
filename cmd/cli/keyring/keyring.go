// Package keyring wraps the system keychain via go-keyring.
//
// Supported backends:
//   - Linux: Secret Service (D-Bus) preferred, kernel keyring (KeyCtl)
//     fallback when no D-Bus session bus is available.
//   - macOS Keychain / Windows Credential Manager via go-keyring.
//
// Open() returns ErrKeyringUnavailable when no persistent backend can be
// initialized. Callers that tolerate secret loss (e.g. `serve`) may fall
// back to NewMemStore(); callers that must persist (`keyring set`) should
// surface the error to the user instead of silently storing in MemStore.
package keyring

import (
	"encoding/base64"
	"errors"
	"sync"

	gkr "github.com/zalando/go-keyring"
)

// ErrKeyringUnavailable is returned by Open when no persistent keyring
// backend can be initialized.
var ErrKeyringUnavailable = errors.New("keyring: no usable backend available")

// Store wraps a persistent system backend. Methods accept a service name
// so callers (including WASM plugins) can access keys under different
// namespaces.
type Store struct {
	backend backend
}

// backend is the storage interface satisfied by each concrete backend
// (Secret Service via zalando, Linux kernel keyring, native keychain on
// macOS/Windows).
type backend interface {
	Get(service, key string) (string, error)
	Set(service, key, value string) error
	Delete(service, key string) error
}

// Open returns a persistent Store. On Linux it prefers Secret Service
// (D-Bus), falling back to the user kernel keyring (KeyCtl). On macOS /
// Windows it uses the native keychain. Returns ErrKeyringUnavailable when
// no persistent backend can be initialized.
func Open() (*Store, error) {
	b, err := openBackend()
	if err != nil {
		return nil, err
	}
	return &Store{backend: b}, nil
}

// HasSupport reports whether a persistent keyring backend is available.
func HasSupport() bool {
	_, err := Open()
	return err == nil
}

func (s *Store) Get(service, key string) (string, error) {
	v, err := s.backend.Get(service, key)
	if err != nil {
		if errors.Is(err, gkr.ErrNotFound) {
			return "", nil
		}
		return "", err
	}
	return v, nil
}

func (s *Store) Set(service, key, value string) error {
	if value == "" {
		return s.backend.Delete(service, key)
	}
	return s.backend.Set(service, key, value)
}

func (s *Store) Delete(service, key string) error {
	err := s.backend.Delete(service, key)
	if errors.Is(err, gkr.ErrNotFound) {
		return nil
	}
	return err
}

// secretServiceBackend wraps zalando/go-keyring (D-Bus on Linux,
// Keychain on macOS, Credential Manager on Windows).
type secretServiceBackend struct{}

func (secretServiceBackend) Get(service, key string) (string, error) {
	return gkr.Get(service, key)
}

func (secretServiceBackend) Set(service, key, value string) error {
	return gkr.Set(service, key, value)
}

func (secretServiceBackend) Delete(service, key string) error {
	return gkr.Delete(service, key)
}

// b64Encode/b64Decode are used by the KeyCtl backend to survive binary
// payloads safely across the kernel keyring API (which expects byte
// slices; we keep parity with hdspace-models/credential encoding too).
func b64Encode(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

func b64Decode(s string) (string, error) {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// MemStore is an in-memory keyring fallback for when the system keychain
// is unavailable (no D-Bus on Linux, headless environments, etc.).
// Secrets do NOT persist across process restarts.
type MemStore struct {
	mu   sync.RWMutex
	keys map[string]string // "service/key" → value
}

// NewMemStore creates an in-memory keyring.
func NewMemStore() *MemStore {
	return &MemStore{keys: make(map[string]string)}
}

func (m *MemStore) gk(service, key string) string { return service + "/" + key }

func (m *MemStore) Get(service, key string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.keys[m.gk(service, key)]
	if !ok {
		return "", nil
	}
	return v, nil
}

func (m *MemStore) Set(service, key, value string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if value == "" {
		delete(m.keys, m.gk(service, key))
	} else {
		m.keys[m.gk(service, key)] = value
	}
	return nil
}

func (m *MemStore) Delete(service, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.keys, m.gk(service, key))
	return nil
}