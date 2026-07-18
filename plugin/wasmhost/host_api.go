// Package wasmhost provides shared WASM infrastructure for plugin hosts.
// Both the CLI plugin system (plugin/cli/wasm) and the Agent plugin system
// (plugin/agent/wasm) use this package for:
//
//   - Host exports (keyring_get/set/delete, http_request, log_info/warn/error)
//   - ABI helpers (pk, up, readPacked)
//   - Host API interfaces (Keyring, HTTPClient, Logger)
package wasmhost

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

// Logger abstracts structured logging for WASM plugins.
type Logger interface {
	Info(msg string)
	Warn(msg string)
	Error(msg string)
}
