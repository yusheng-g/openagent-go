package rest

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	openagent "github.com/yusheng-g/openagent-go"
)

// stubModel is a minimal openagent.Model with a configurable ContextWindow
// and a recorded modelID, used to verify which model a team run uses.
type stubModel struct {
	name    string
	cw      int
	called  int
	lastReq openagent.ChatCompletionRequest
}

func (m *stubModel) ChatCompletion(ctx context.Context, req openagent.ChatCompletionRequest) (*openagent.ChatCompletionResponse, error) {
	m.called++
	m.lastReq = req
	return &openagent.ChatCompletionResponse{
		Choices: []openagent.Choice{{
			Message:      openagent.Message{Role: openagent.RoleAssistant, Content: "ok"},
			FinishReason: "stop",
		}},
	}, nil
}

func (m *stubModel) ChatCompletionStream(ctx context.Context, req openagent.ChatCompletionRequest) (openagent.StreamReader, error) {
	m.called++
	m.lastReq = req
	return nil, nil
}

func (m *stubModel) ContextWindow() int { return m.cw }

// newTeamTestHandler builds a TeamHandler with one template agent on modelA
// and registers modelA + modelB in the registry.
func newTeamTestHandler(t *testing.T, modelA, modelB *stubModel) *TeamHandler {
	t.Helper()
	agent := openagent.NewAgent("solo",
		openagent.WithModel(modelA),
		openagent.WithSystemPrompts("test"),
		openagent.WithMaxTurns(1),
	)
	h := NewTeamHandler(nil, TeamAgentTemplate{Name: "solo", Description: "d", Agent: agent})
	h.RegisterModel("model-a", modelA, "p")
	h.RegisterModel("model-b", modelB, "p")
	return h
}

// readSSEDone reads SSE events until a "done" or "error" line is seen.
func readSSEDone(body string) string {
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "data: ") {
			return strings.TrimPrefix(line, "data: ")
		}
	}
	return ""
}

// TestTeamHandleChatModelOverride verifies that ChatRequest.ModelID selects
// the registered model and overrides the team agent's default for the run.
func TestTeamHandleChatModelOverride(t *testing.T) {
	modelA := &stubModel{name: "A", cw: 8000}
	modelB := &stubModel{name: "B", cw: 16000}
	h := newTeamTestHandler(t, modelA, modelB)

	mux := http.NewServeMux()
	h.Register(mux)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	// Create a team session.
	createResp, err := http.Post(srv.URL+"/team/sessions", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	defer createResp.Body.Close()
	var info struct {
		ID string `json:"sessionId"`
	}
	if err := json.NewDecoder(createResp.Body).Decode(&info); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	if info.ID == "" {
		t.Fatal("no session id")
	}

	// Chat selecting model-b.
	chatResp, err := http.Post(srv.URL+"/team/sessions/"+info.ID+"/chat",
		"application/json",
		strings.NewReader(`{"message":"hi","modelId":"model-b","provider":"p"}`))
	if err != nil {
		t.Fatalf("chat: %v", err)
	}
	defer chatResp.Body.Close()
	if chatResp.StatusCode != http.StatusOK {
		t.Fatalf("chat status %d", chatResp.StatusCode)
	}
	body, _ := io.ReadAll(chatResp.Body)
	readSSEDone(string(body))

	if modelB.called == 0 {
		t.Error("model-b was not used for the run — frontend model selection has no effect")
	}
}

// TestTeamGetSessionContextWindow verifies that GET /team/sessions/{id}
// returns a non-zero ContextWindow derived from the session's model.
func TestTeamGetSessionContextWindow(t *testing.T) {
	modelA := &stubModel{name: "A", cw: 8000}
	modelB := &stubModel{name: "B", cw: 16000}
	h := newTeamTestHandler(t, modelA, modelB)

	mux := http.NewServeMux()
	h.Register(mux)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	// Create + chat selecting model-b so meta is persisted.
	createResp, err := http.Post(srv.URL+"/team/sessions", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	var info struct {
		ID string `json:"sessionId"`
	}
	json.NewDecoder(createResp.Body).Decode(&info)
	createResp.Body.Close()

	chatResp, err := http.Post(srv.URL+"/team/sessions/"+info.ID+"/chat",
		"application/json",
		strings.NewReader(`{"message":"hi","modelId":"model-b","provider":"p"}`))
	if err != nil {
		t.Fatalf("chat: %v", err)
	}
	io.ReadAll(chatResp.Body)
	chatResp.Body.Close()

	// GET session detail.
	getResp, err := http.Get(srv.URL + "/team/sessions/" + info.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer getResp.Body.Close()
	var detail struct {
		ContextWindow int            `json:"contextWindow"`
		Meta          map[string]any `json:"_meta"`
	}
	if err := json.NewDecoder(getResp.Body).Decode(&detail); err != nil {
		t.Fatalf("decode detail: %v", err)
	}
	if detail.ContextWindow != 16000 {
		t.Errorf("ContextWindow = %d, want 16000 (model-b's window)", detail.ContextWindow)
	}
	if got, _ := detail.Meta["modelId"].(string); got != "model-b" {
		t.Errorf("meta modelId = %v, want model-b (stale ModelID bug)", detail.Meta["modelId"])
	}
}

// TestTeamHandleChatUnknownModelFallback verifies that an unregistered modelId
// falls back to h.model (the first template agent's model) rather than failing
// or silently dropping the run. Covers the model == nil → h.model branch in
// handleChat (team.go:274-277).
func TestTeamHandleChatUnknownModelFallback(t *testing.T) {
	modelA := &stubModel{name: "A", cw: 8000}
	modelB := &stubModel{name: "B", cw: 16000}
	h := newTeamTestHandler(t, modelA, modelB)

	mux := http.NewServeMux()
	h.Register(mux)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	createResp, err := http.Post(srv.URL+"/team/sessions", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	var info struct {
		ID string `json:"sessionId"`
	}
	json.NewDecoder(createResp.Body).Decode(&info)
	createResp.Body.Close()

	// Chat selecting a modelId that was never registered (with its provider).
	chatResp, err := http.Post(srv.URL+"/team/sessions/"+info.ID+"/chat",
		"application/json",
		strings.NewReader(`{"message":"hi","modelId":"no-such-model","provider":"p"}`))
	if err != nil {
		t.Fatalf("chat: %v", err)
	}
	defer chatResp.Body.Close()
	if chatResp.StatusCode != http.StatusOK {
		t.Fatalf("chat status %d, want 200 (unknown model should fall back, not error)", chatResp.StatusCode)
	}
	body, _ := io.ReadAll(chatResp.Body)
	readSSEDone(string(body))

	// h.model is modelA (first template), so the fallback run must use modelA
	// and must NOT touch modelB.
	if modelA.called == 0 {
		t.Error("modelA (h.model fallback) was not used for an unknown modelId")
	}
	if modelB.called != 0 {
		t.Error("modelB was invoked during fallback — lookupModel leaked across registry entries")
	}

	// The run fell back to h.model, but meta still records the requested id.
	// GET must report ContextWindow from the fallback model (modelA, 8000),
	// proving fillDetail resolves the effective model, not the stale request id.
	getResp, err := http.Get(srv.URL + "/team/sessions/" + info.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer getResp.Body.Close()
	var detail struct {
		ContextWindow int            `json:"contextWindow"`
		Meta          map[string]any `json:"_meta"`
	}
	if err := json.NewDecoder(getResp.Body).Decode(&detail); err != nil {
		t.Fatalf("decode detail: %v", err)
	}
	if detail.ContextWindow != 8000 {
		t.Errorf("ContextWindow = %d, want 8000 (fallback modelA's window)", detail.ContextWindow)
	}
}

// TestTeamHandleChatModelIDOnlyNoProvider verifies the provider-less lookup
// path: when only modelId is sent (provider == ""), lookupModel scans via
// strings.HasSuffix(":"+modelID) and must resolve p:model-b → modelB. Covers
// the for-loop branch in lookupModel (team.go:96-103).
func TestTeamHandleChatModelIDOnlyNoProvider(t *testing.T) {
	modelA := &stubModel{name: "A", cw: 8000}
	modelB := &stubModel{name: "B", cw: 16000}
	h := newTeamTestHandler(t, modelA, modelB)

	mux := http.NewServeMux()
	h.Register(mux)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	createResp, err := http.Post(srv.URL+"/team/sessions", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	var info struct {
		ID string `json:"sessionId"`
	}
	json.NewDecoder(createResp.Body).Decode(&info)
	createResp.Body.Close()

	// Chat selecting model-b with NO provider — exercises the HasSuffix scan.
	chatResp, err := http.Post(srv.URL+"/team/sessions/"+info.ID+"/chat",
		"application/json",
		strings.NewReader(`{"message":"hi","modelId":"model-b"}`))
	if err != nil {
		t.Fatalf("chat: %v", err)
	}
	defer chatResp.Body.Close()
	if chatResp.StatusCode != http.StatusOK {
		t.Fatalf("chat status %d", chatResp.StatusCode)
	}
	body, _ := io.ReadAll(chatResp.Body)
	readSSEDone(string(body))

	if modelB.called == 0 {
		t.Error("model-b was not resolved by the provider-less HasSuffix scan")
	}
	if modelA.called != 0 {
		t.Error("modelA was invoked — provider-less scan should resolve model-b, not fall back")
	}
}
