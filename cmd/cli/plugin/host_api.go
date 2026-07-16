package plugin

// ── Host API interfaces exposed to WASM plugins ──

type Keyring interface {
	Get(service, key string) (string, error)
	Set(service, key, value string) error
	Delete(service, key string) error
}

type HTTPClient interface {
	Do(method, url string, headers map[string]string, body []byte) (status int, respBody []byte, err error)
}

type Logger interface {
	Info(msg string)
	Warn(msg string)
	Error(msg string)
}
