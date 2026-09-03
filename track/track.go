// Package track reports events to the Huawei Cloud martech tracking platform
// via a Base64-JSON-form-urlencoded POST protocol.  Currently it reports
// session lifecycle events (create / close / delete); the package is
// structured so other event types can be added without changing the
// reporting infrastructure.
//
// The package is self-contained: when EventPostUrl is empty (the default for
// bare `go build`), Init is a no-op and all Report* calls return immediately
// without making network requests.  Only builds with ldflags-injected values
// (e.g. `make x86`) emit events.
//
// Observer implements openagent.SessionObserver, so the track package can be
// wired into the ACP/CLI servers via SetSessionObservers without those
// servers importing track directly.  MultiSessionObserver is available for
// multi-observer fan-out (e.g. track + OpenTelemetry).
package track

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	openagent "github.com/yusheng-g/openagent-go"
	"github.com/yusheng-g/openagent-go/version"
)

// Package-level configuration, injected via ldflags at build time.
// When EventPostUrl is empty, the package is fully disabled (no-op).
var (
	EventPostUrl    string    // tracking platform endpoint
	AppID           string    // tracking platform app ID
	EventSkipVerify = "false" // skip TLS certificate verification ("true" | "false")
)

// HTTP timeout for the tracking POST.  Bounded so a hung endpoint cannot
// stall the caller indefinitely.
const trackTimeout = 5 * time.Second

var (
	mu       sync.Mutex
	observer *Observer
)

func init() {
	observer = &Observer{}
}

// EventType constants — the "event" field in EventReq.
const (
	EventSessionCreate = "Session_Create"
	EventSessionClose  = "Session_Close"
	EventSessionDelete = "Session_Delete"
)

// EntryPoint constants — the "entry_point" property in EventReq.
const (
	EntryPointACP  = "acp"
	EntryPointCLI  = "cli"
	// EntryPointREST is reserved for future use.  REST sessions are not
	// wired to the observer because REST does not serve conversations in
	// production (dialogue goes through ACP).
	EntryPointREST = "rest"
)

// EventReq is the contract with the tracking platform.
// Field names and json tags MUST NOT change — only the field values.
type EventReq struct {
	AnonymousId string     `json:"anonymousId"`
	DistinctId  string     `json:"distinctId"`
	Event       string     `json:"event"`
	Time        int64      `json:"time"`
	Type        string     `json:"type"`
	Properties  properties `json:"properties"`
}

// properties is the free-form JSON sub-object for custom fields.
type properties struct {
	Source      string `json:"source"`
	Version     string `json:"version"`
	SessionID   string `json:"session_id"`
	EntryPoint  string `json:"entry_point"`
	SessionMode string `json:"session_mode"`
	DurationMs  int64  `json:"duration_ms"`
	FailReason  string `json:"fail_reason"`
	Time        string `json:"time"`
}

// SessionParams is the input for building an EventReq from a session event.
type SessionParams struct {
	SessionID   string
	EntryPoint  string
	SessionMode string
	DurationMs  int64
	FailReason  string
}

// eventHttpClient is set once by Init and read by ReportEvent.  Init must
// complete before any goroutine calls ReportEvent — this happens-before
// constraint is satisfied by calling Init during single-threaded startup in
// cmd/cli/main.go before the server begins accepting requests.
var eventHttpClient *http.Client

// Init constructs the HTTP client used for tracking POSTs.
// Safe to call multiple times.  No-op when EventPostUrl is empty.
// Must be called once during startup, before any concurrent ReportEvent calls.
func Init() {
	if EventPostUrl == "" {
		return
	}
	if strings.EqualFold(EventSkipVerify, "true") {
		slog.Warn("track: TLS certificate verification disabled — insecure")
	}
	mu.Lock()
	defer mu.Unlock()
	if eventHttpClient != nil {
		return
	}
	eventHttpClient = &http.Client{
		Timeout: trackTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion:         tls.VersionTLS12,
				InsecureSkipVerify: strings.EqualFold(EventSkipVerify, "true"),
			},
		},
	}
}

// BuildEvent constructs an EventReq from session-scoped params.
func BuildEvent(event string, params SessionParams) EventReq {
	now := time.Now()
	return EventReq{
		AnonymousId: params.SessionID,
		DistinctId:  params.SessionID,
		Event:       event,
		Time:        now.UnixMilli(),
		Type:        "track",
		Properties: properties{
			Source:      version.Name,
			Version:     version.Version,
			SessionID:   params.SessionID,
			EntryPoint:  params.EntryPoint,
			SessionMode: params.SessionMode,
			DurationMs:  params.DurationMs,
			FailReason:  params.FailReason,
			Time:        now.UTC().Format(time.RFC3339Nano),
		},
	}
}

// ReportEvent sends a single event to the tracking platform.
// Panic-safe, non-blocking (bounded by trackTimeout), logs errors via slog.
// No-op when EventPostUrl is empty or the client was never initialized.
func ReportEvent(ctx context.Context, eventReq EventReq) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("track: event reporting panic", "panic", r)
		}
	}()

	if EventPostUrl == "" || eventHttpClient == nil {
		return
	}

	events := []EventReq{eventReq}
	base64Data, err := encodeToBase64(events)
	if err != nil {
		slog.Error("track: failed to encode event data", "error", err)
		return
	}

	formData := url.Values{
		"data":  []string{base64Data},
		"appid": []string{AppID},
		"debug": []string{"0"},
		"gzip":  []string{"0"},
	}
	if err := sendPostRequest(ctx, formData); err != nil {
		slog.Error("track: failed to send event data", "error", err)
	}
}

// sendPostRequest POSTs form-urlencoded data to the tracking endpoint.
func sendPostRequest(ctx context.Context, formData url.Values) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, EventPostUrl, bytes.NewBufferString(formData.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := eventHttpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Warn("track: failed to close response body", "error", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("request failed with status code: %d", resp.StatusCode)
	}
	return nil
}

// encodeToBase64 marshals data to JSON and then Base64-encodes the result.
func encodeToBase64(data interface{}) (string, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(jsonData), nil
}

// ── SessionObserver implementation ──

// Observer implements openagent.SessionObserver, adapting session lifecycle
// events to HTTP tracking reports.  It is the bridge between the
// SessionObserver interface and the Report* functions in this package.
type Observer struct{}

// GetObserver returns the package-level Observer singleton.
// Safe to call before Init; the Observer methods are no-op when disabled.
func GetObserver() *Observer {
	return observer
}

func (o *Observer) OnSessionCreate(ctx context.Context, e openagent.SessionLifecycleEvent) {
	ReportEvent(ctx, BuildEvent(EventSessionCreate, SessionParams{
		SessionID:   e.SessionID,
		EntryPoint:  e.EntryPoint,
		SessionMode: e.SessionMode,
	}))
}

func (o *Observer) OnSessionClose(ctx context.Context, e openagent.SessionLifecycleEvent) {
	failReason := ""
	if e.Err != nil {
		failReason = e.Err.Error()
	}
	ReportEvent(ctx, BuildEvent(EventSessionClose, SessionParams{
		SessionID:   e.SessionID,
		EntryPoint:  e.EntryPoint,
		SessionMode: e.SessionMode,
		DurationMs:  e.DurationMs,
		FailReason:  failReason,
	}))
}

func (o *Observer) OnSessionDelete(ctx context.Context, e openagent.SessionLifecycleEvent) {
	failReason := ""
	if e.Err != nil {
		failReason = e.Err.Error()
	}
	ReportEvent(ctx, BuildEvent(EventSessionDelete, SessionParams{
		SessionID:   e.SessionID,
		EntryPoint:  e.EntryPoint,
		SessionMode: e.SessionMode,
		DurationMs:  e.DurationMs,
		FailReason:  failReason,
	}))
}
