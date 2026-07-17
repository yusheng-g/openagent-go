// Package keyring wraps the system keychain via go-keyring.
//
// Supported: Linux Secret Service (D-Bus), macOS Keychain, Windows Credential Manager.
package keyring

import (
	"errors"
	"sync"

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
	if !ok { return "", nil }
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
