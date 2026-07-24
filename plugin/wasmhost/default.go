package wasmhost

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
)

// NewHostAPI constructs a HostAPI with the given keyring and sensible
// defaults for HTTP (net/http) and logging (standard log adapter).
func NewHostAPI(kr Keyring) *HostAPI {
	return &HostAPI{
		Keyring: kr,
		HTTP:    &defaultHTTPClient{client: http.DefaultClient},
		Logger:  &logAdapter{},
	}
}

// defaultHTTPClient implements HTTPClient via net/http.
type defaultHTTPClient struct{ client *http.Client }

func (c *defaultHTTPClient) Do(method, url string, headers map[string]string, body []byte) (int, []byte, error) {
	req, _ := http.NewRequest(method, url, bytes.NewReader(body))
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, respBody, nil
}

// logAdapter implements Logger by forwarding to the standard log package.
type logAdapter struct{}

func (l *logAdapter) Info(msg string)  { slog.Info(msg, "source", "plugin") }
func (l *logAdapter) Warn(msg string)  { slog.Warn(msg, "source", "plugin") }
func (l *logAdapter) Error(msg string) { slog.Error(msg, "source", "plugin") }
