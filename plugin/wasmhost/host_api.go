// Package wasmhost provides shared WASM infrastructure for plugin hosts.
// Both the CLI plugin system (plugin/cli/wasm) and the Agent plugin system
// (plugin/agent/wasm) use this package for:
//
//   - Host exports (keyring_get/set/delete, http_request, fs_read/write/readdir, log_*)
//   - ABI helpers (Pack, Unpack, ReadPacked, ReadString, WriteString)
//   - Host API interfaces (Keyring, HTTPClient, FS, Logger)
package wasmhost

import "os"

// Keyring abstracts the system keychain for WASM plugins.
type Keyring interface {
	Get(service, key string) (string, error)
	Set(service, key, value string) error
	Delete(service, key string) error
}

// HTTPClient abstracts HTTP outbound calls for WASM plugins.
type HTTPClient interface {
	Do(method, url string, headers map[string]string, body []byte) (status int, respBody []byte, err error)
}

// FS abstracts filesystem access for WASM plugins.
// Paths are relative or absolute; the implementation decides the sandbox boundary.
type FS interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte) error
	ReadDir(path string) ([]os.DirEntry, error)
}

// Logger abstracts structured logging for WASM plugins.
type Logger interface {
	Info(msg string)
	Warn(msg string)
	Error(msg string)
}

// HostAPI bundles host-provided capabilities available to WASM plugins
// via the "host" module.
type HostAPI struct {
	Keyring Keyring
	HTTP    HTTPClient
	FS      FS
	Logger  Logger
}
