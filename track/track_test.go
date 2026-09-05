package track

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	openagent "github.com/yusheng-g/openagent-go"
)

// ── Unit tests ──

func TestBuildEvent_FieldMapping(t *testing.T) {
	params := SessionParams{
		SessionID:   "sess-123",
		EntryPoint:  EntryPointACP,
		SessionMode: "manual",
		DurationMs:  5000,
		FailReason:  "test error",
	}
	evt := BuildEvent(EventSessionCreate, params)

	if evt.Event != EventSessionCreate {
		t.Errorf("Event = %q, want %q", evt.Event, EventSessionCreate)
	}
	if evt.AnonymousId != "sess-123" {
		t.Errorf("AnonymousId = %q, want sess-123", evt.AnonymousId)
	}
	if evt.DistinctId != "sess-123" {
		t.Errorf("DistinctId = %q, want sess-123", evt.DistinctId)
	}
	if evt.Type != "track" {
		t.Errorf("Type = %q, want track", evt.Type)
	}
	if evt.Time <= 0 {
		t.Error("Time should be positive (UnixMilli)")
	}
	if evt.Properties.SessionID != "sess-123" {
		t.Errorf("Properties.SessionID = %q, want sess-123", evt.Properties.SessionID)
	}
	if evt.Properties.EntryPoint != "acp" {
		t.Errorf("Properties.EntryPoint = %q, want acp", evt.Properties.EntryPoint)
	}
	if evt.Properties.SessionMode != "manual" {
		t.Errorf("Properties.SessionMode = %q, want manual", evt.Properties.SessionMode)
	}
	if evt.Properties.DurationMs != 5000 {
		t.Errorf("Properties.DurationMs = %d, want 5000", evt.Properties.DurationMs)
	}
	if evt.Properties.FailReason != "test error" {
		t.Errorf("Properties.FailReason = %q, want 'test error'", evt.Properties.FailReason)
	}
	if evt.Properties.Time == "" {
		t.Error("Properties.Time should not be empty (RFC3339Nano)")
	}
}

func TestBuildEvent_AllThreeEventTypes(t *testing.T) {
	params := SessionParams{SessionID: "s1", EntryPoint: EntryPointCLI}

	create := BuildEvent(EventSessionCreate, params)
	closeEvt := BuildEvent(EventSessionClose, params)
	deleteEvt := BuildEvent(EventSessionDelete, params)

	if create.Event != "Session_Create" {
		t.Errorf("create event = %q, want Session_Create", create.Event)
	}
	if closeEvt.Event != "Session_Close" {
		t.Errorf("close event = %q, want Session_Close", closeEvt.Event)
	}
	if deleteEvt.Event != "Session_Delete" {
		t.Errorf("delete event = %q, want Session_Delete", deleteEvt.Event)
	}
}

func TestEncodeToBase64_RoundTrip(t *testing.T) {
	original := []EventReq{
		BuildEvent(EventSessionCreate, SessionParams{SessionID: "s1", EntryPoint: "acp"}),
	}
	encoded, err := encodeToBase64(original)
	if err != nil {
		t.Fatalf("encodeToBase64 failed: %v", err)
	}

	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("base64 decode failed: %v", err)
	}

	var result []EventReq
	if err := json.Unmarshal(decoded, &result); err != nil {
		t.Fatalf("json unmarshal failed: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("decoded length = %d, want 1", len(result))
	}
	if result[0].Event != original[0].Event {
		t.Errorf("decoded event = %q, want %q", result[0].Event, original[0].Event)
	}
	if result[0].Properties.SessionID != original[0].Properties.SessionID {
		t.Errorf("decoded session_id = %q, want %q",
			result[0].Properties.SessionID, original[0].Properties.SessionID)
	}
}

func TestReportEvent_NoOpWhenUrlEmpty(t *testing.T) {
	// Save and restore state.
	savedUrl := EventPostUrl
	savedClient := eventHttpClient
	defer func() {
		EventPostUrl = savedUrl
		eventHttpClient = savedClient
	}()

	EventPostUrl = ""
	eventHttpClient = nil

	// Should return immediately without panicking or making network calls.
	evt := BuildEvent(EventSessionCreate, SessionParams{SessionID: "s1"})
	ReportEvent(context.Background(), evt) // must not block or panic
}

func TestReportEvent_NoOpWhenClientNil(t *testing.T) {
	savedUrl := EventPostUrl
	savedClient := eventHttpClient
	defer func() {
		EventPostUrl = savedUrl
		eventHttpClient = savedClient
	}()

	EventPostUrl = "http://example.com/track"
	eventHttpClient = nil // client never initialized

	evt := BuildEvent(EventSessionCreate, SessionParams{SessionID: "s1"})
	ReportEvent(context.Background(), evt) // must not block or panic
}

func TestReportEvent_PanicSafe(t *testing.T) {
	savedUrl := EventPostUrl
	savedClient := eventHttpClient
	defer func() {
		EventPostUrl = savedUrl
		eventHttpClient = savedClient
	}()

	// Set up a client pointing to an invalid URL to trigger an error path.
	EventPostUrl = "http://invalid-url-that-does-not-exist.invalid/track"
	eventHttpClient = &http.Client{Timeout: 1 * time.Second}

	// Should not panic even if the request fails.
	evt := BuildEvent(EventSessionCreate, SessionParams{SessionID: "s1"})
	ReportEvent(context.Background(), evt)
}

func TestInit_NoOpWhenUrlEmpty(t *testing.T) {
	savedUrl := EventPostUrl
	savedClient := eventHttpClient
	defer func() {
		EventPostUrl = savedUrl
		eventHttpClient = savedClient
	}()

	EventPostUrl = ""
	eventHttpClient = nil
	Init()
	if eventHttpClient != nil {
		t.Error("Init should not create client when EventPostUrl is empty")
	}
}

func TestInit_Idempotent(t *testing.T) {
	savedUrl := EventPostUrl
	savedClient := eventHttpClient
	defer func() {
		EventPostUrl = savedUrl
		eventHttpClient = savedClient
	}()

	EventPostUrl = "http://localhost:9999/track"
	eventHttpClient = nil
	Init()
	first := eventHttpClient
	Init() // second call should not replace the client
	if eventHttpClient != first {
		t.Error("Init should be idempotent (same client pointer)")
	}
}

// ── Observer tests ──

func TestObserver_DelegatesToReportEvent(t *testing.T) {
	savedUrl := EventPostUrl
	savedClient := eventHttpClient
	defer func() {
		EventPostUrl = savedUrl
		eventHttpClient = savedClient
	}()

	// Use httptest to verify the Observer methods produce correct events.
	var receivedBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		receivedBody = string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	EventPostUrl = server.URL
	AppID = "test-app"
	eventHttpClient = server.Client()

	obs := &Observer{}
	obs.OnSessionCreate(context.Background(), openagent.SessionLifecycleEvent{
		SessionID:   "sess-obs-1",
		EntryPoint:  EntryPointACP,
		SessionMode: "plan",
	})

	// Decode the received form data.
	form, err := url.ParseQuery(receivedBody)
	if err != nil {
		t.Fatalf("ParseQuery failed: %v", err)
	}
	if form.Get("appid") != "test-app" {
		t.Errorf("appid = %q, want test-app", form.Get("appid"))
	}

	// Decode the Base64 JSON event.
	data := form.Get("data")
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		t.Fatalf("base64 decode failed: %v", err)
	}
	var events []EventReq
	if err := json.Unmarshal(decoded, &events); err != nil {
		t.Fatalf("json unmarshal failed: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Event != EventSessionCreate {
		t.Errorf("event = %q, want %q", events[0].Event, EventSessionCreate)
	}
	if events[0].Properties.SessionID != "sess-obs-1" {
		t.Errorf("session_id = %q, want sess-obs-1", events[0].Properties.SessionID)
	}
	if events[0].Properties.SessionMode != "plan" {
		t.Errorf("session_mode = %q, want plan", events[0].Properties.SessionMode)
	}
}

func TestObserver_OnSessionClose_IncludesDuration(t *testing.T) {
	savedUrl := EventPostUrl
	savedClient := eventHttpClient
	defer func() {
		EventPostUrl = savedUrl
		eventHttpClient = savedClient
	}()

	var receivedBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		receivedBody = string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	EventPostUrl = server.URL
	AppID = "test-app"
	eventHttpClient = server.Client()

	obs := &Observer{}
	obs.OnSessionClose(context.Background(), openagent.SessionLifecycleEvent{
		SessionID:  "sess-obs-2",
		EntryPoint: EntryPointCLI,
		DurationMs: 12345,
	})

	form, _ := url.ParseQuery(receivedBody)
	decoded, _ := base64.StdEncoding.DecodeString(form.Get("data"))
	var events []EventReq
	if err := json.Unmarshal(decoded, &events); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if events[0].Event != EventSessionClose {
		t.Errorf("event = %q, want %q", events[0].Event, EventSessionClose)
	}
	if events[0].Properties.DurationMs != 12345 {
		t.Errorf("duration_ms = %d, want 12345", events[0].Properties.DurationMs)
	}
}

func TestObserver_OnSessionDelete_IncludesFailReason(t *testing.T) {
	savedUrl := EventPostUrl
	savedClient := eventHttpClient
	defer func() {
		EventPostUrl = savedUrl
		eventHttpClient = savedClient
	}()

	var receivedBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		receivedBody = string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	EventPostUrl = server.URL
	AppID = "test-app"
	eventHttpClient = server.Client()

	obs := &Observer{}
	obs.OnSessionDelete(context.Background(), openagent.SessionLifecycleEvent{
		SessionID:  "sess-obs-3",
		EntryPoint: EntryPointACP,
		Err:        io.ErrUnexpectedEOF,
	})

	form, _ := url.ParseQuery(receivedBody)
	decoded, _ := base64.StdEncoding.DecodeString(form.Get("data"))
	var events []EventReq
	if err := json.Unmarshal(decoded, &events); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if events[0].Event != EventSessionDelete {
		t.Errorf("event = %q, want %q", events[0].Event, EventSessionDelete)
	}
	if !strings.Contains(events[0].Properties.FailReason, "unexpected EOF") {
		t.Errorf("fail_reason = %q, want it to contain 'unexpected EOF'",
			events[0].Properties.FailReason)
	}
}

// ── Integration test ──

func TestIntegration_ServerReceivesAllThreeEventTypes(t *testing.T) {
	savedUrl := EventPostUrl
	savedClient := eventHttpClient
	savedAppID := AppID
	defer func() {
		EventPostUrl = savedUrl
		eventHttpClient = savedClient
		AppID = savedAppID
	}()

	var receivedPosts []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		receivedPosts = append(receivedPosts, string(body))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	EventPostUrl = server.URL
	AppID = "integration-test"
	eventHttpClient = server.Client()

	ctx := context.Background()
	obs := &Observer{}

	obs.OnSessionCreate(ctx, openagent.SessionLifecycleEvent{
		SessionID:  "s-int-1",
		EntryPoint: EntryPointACP,
		CreatedAt:  time.Now().Add(-10 * time.Second),
	})
	obs.OnSessionClose(ctx, openagent.SessionLifecycleEvent{
		SessionID:  "s-int-1",
		EntryPoint: EntryPointACP,
		DurationMs: 10000,
	})
	obs.OnSessionDelete(ctx, openagent.SessionLifecycleEvent{
		SessionID:  "s-int-2",
		EntryPoint: EntryPointREST,
	})

	if len(receivedPosts) != 3 {
		t.Fatalf("expected 3 POSTs, got %d", len(receivedPosts))
	}

	// Decode each POST and verify the event type.
	wantEvents := []string{EventSessionCreate, EventSessionClose, EventSessionDelete}
	for i, body := range receivedPosts {
		form, err := url.ParseQuery(body)
		if err != nil {
			t.Fatalf("POST %d: ParseQuery failed: %v", i, err)
		}
		if form.Get("appid") != "integration-test" {
			t.Errorf("POST %d: appid = %q, want integration-test", i, form.Get("appid"))
		}
		decoded, _ := base64.StdEncoding.DecodeString(form.Get("data"))
		var events []EventReq
		if err := json.Unmarshal(decoded, &events); err != nil {
			t.Fatalf("POST %d: unmarshal failed: %v", i, err)
		}
		if len(events) != 1 {
			t.Fatalf("POST %d: expected 1 event, got %d", i, len(events))
		}
		if events[0].Event != wantEvents[i] {
			t.Errorf("POST %d: event = %q, want %q", i, events[0].Event, wantEvents[i])
		}
	}
}
