// ACP agent server example — a calculator agent that communicates over stdio.
//
// This agent implements the full openacp.AgentHandler interface and can be
// spawned as a subprocess by any ACP client.
//
//	go build -o /tmp/acp-calc-server ./examples/acp/server/
//	/tmp/acp-calc-server                          # blocks on stdio
//
// Or run the companion client:
//
//	go run ./examples/acp/
package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	openacp "github.com/yusheng-g/openagent-go/acp/sdk"
)

func main() {
	srv := newCalcServer()
	server := openacp.NewServer("acp-calculator", "1.0.0", srv)
	log.SetPrefix("[acp-server] ")
	log.SetFlags(log.Ltime)
	log.Println("starting on stdio...")
	if err := server.Run(context.Background()); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}

type calcServer struct {
	mu       sync.RWMutex
	sessions map[string]*sessionState
	nextID   int
}

type sessionState struct {
	id        string
	cwd       string
	title     string
	updatedAt time.Time
}

func newCalcServer() *calcServer {
	return &calcServer{sessions: make(map[string]*sessionState)}
}

// ── AgentHandler ──

func (s *calcServer) OnInitialize(ctx context.Context, req openacp.InitializeRequest) (*openacp.InitializeResponse, error) {
	log.Printf("initialize: proto=%d terminal=%v", req.ProtocolVersion, req.ClientCapabilities.Terminal)
	return &openacp.InitializeResponse{
		ProtocolVersion: 1,
		AgentCapabilities: openacp.AgentCapabilities{
			LoadSession: true,
			SessionCapabilities: openacp.SessionCapabilities{
				Close:  &openacp.SessionCloseCapabilities{},
				Delete: &openacp.SessionDeleteCapabilities{},
				List:   &openacp.SessionListCapabilities{},
				Resume: &openacp.SessionResumeCapabilities{},
			},
		},
	}, nil
}

func (s *calcServer) OnAuthenticate(ctx context.Context, req openacp.AuthenticateRequest) (*openacp.AuthenticateResponse, error) {
	return &openacp.AuthenticateResponse{}, nil
}

func (s *calcServer) OnNewSession(ctx context.Context, req openacp.NewSessionRequest) (*openacp.NewSessionResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	id := fmt.Sprintf("sess_%d", s.nextID)
	st := &sessionState{id: id, cwd: req.Cwd, updatedAt: time.Now()}
	s.sessions[id] = st
	log.Printf("new session: %s (cwd=%s)", id, req.Cwd)
	return &openacp.NewSessionResponse{SessionID: id}, nil
}

func (s *calcServer) OnLoadSession(ctx context.Context, req openacp.LoadSessionRequest, sender openacp.SessionEventSender) (*openacp.LoadSessionResponse, error) {
	s.mu.RLock()
	st, ok := s.sessions[req.SessionID]
	s.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("session not found: %s", req.SessionID)
	}
	st.updatedAt = time.Now()
	log.Printf("load session: %s", req.SessionID)
	return &openacp.LoadSessionResponse{}, nil
}

func (s *calcServer) OnResumeSession(ctx context.Context, req openacp.ResumeSessionRequest) (*openacp.ResumeSessionResponse, error) {
	s.mu.RLock()
	_, ok := s.sessions[req.SessionID]
	s.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("session not found: %s", req.SessionID)
	}
	return &openacp.ResumeSessionResponse{}, nil
}

func (s *calcServer) OnCloseSession(ctx context.Context, req openacp.CloseSessionRequest) (*openacp.CloseSessionResponse, error) {
	s.mu.Lock()
	delete(s.sessions, req.SessionID)
	s.mu.Unlock()
	log.Printf("close session: %s", req.SessionID)
	return &openacp.CloseSessionResponse{}, nil
}

func (s *calcServer) OnDeleteSession(ctx context.Context, req openacp.DeleteSessionRequest) (*openacp.DeleteSessionResponse, error) {
	s.mu.Lock()
	delete(s.sessions, req.SessionID)
	s.mu.Unlock()
	log.Printf("delete session: %s", req.SessionID)
	return &openacp.DeleteSessionResponse{}, nil
}

func (s *calcServer) OnListSessions(ctx context.Context, req openacp.ListSessionsRequest) (*openacp.ListSessionsResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var sessions []openacp.SessionInfo
	for _, st := range s.sessions {
		sessions = append(sessions, openacp.SessionInfo{
			SessionID: st.id, Cwd: st.cwd, Title: st.title,
			UpdatedAt: st.updatedAt.Format(time.RFC3339),
		})
	}
	log.Printf("list sessions: %d found", len(sessions))
	return &openacp.ListSessionsResponse{Sessions: sessions}, nil
}

func (s *calcServer) OnListMessages(ctx context.Context, req openacp.ListMessagesRequest) (*openacp.ListMessagesResponse, error) {
	// The example stores no conversation history; report an empty list.
	return &openacp.ListMessagesResponse{Messages: []openacp.Message{}}, nil
}

func (s *calcServer) OnListConfigOptions(ctx context.Context, req openacp.ListConfigOptionsRequest) (*openacp.ListConfigOptionsResponse, error) {
	// The example exposes no session config options.
	return &openacp.ListConfigOptionsResponse{ConfigOptions: []openacp.SessionConfigOption{}}, nil
}

func (s *calcServer) OnSetSessionMode(ctx context.Context, req openacp.SetSessionModeRequest) (*openacp.SetSessionModeResponse, error) {
	return &openacp.SetSessionModeResponse{}, nil
}

func (s *calcServer) OnSetSessionConfigOption(ctx context.Context, req openacp.SetSessionConfigOptionRequest) (*openacp.SetSessionConfigOptionResponse, error) {
	return &openacp.SetSessionConfigOptionResponse{}, nil
}

func (s *calcServer) OnPrompt(ctx context.Context, req openacp.PromptRequest, sender openacp.SessionEventSender) (*openacp.PromptResponse, error) {
	var input string
	for _, b := range req.Prompt {
		if b.Text != "" {
			input = b.Text
			break
		}
	}
	log.Printf("prompt [%s]: %q", req.SessionID, truncate(input, 60))

	s.mu.Lock()
	if st, ok := s.sessions[req.SessionID]; ok {
		st.updatedAt = time.Now()
		st.title = firstLine(input, 40)
	}
	s.mu.Unlock()

	expr, ok := extractExpression(input)
	if ok {
		return s.handleCalculation(ctx, req.SessionID, expr, sender)
	}
	_ = sender.SendAgentMessage(fmt.Sprintf("Received: %s\n", input))
	return &openacp.PromptResponse{StopReason: openacp.StopReasonEndTurn}, nil
}

func (s *calcServer) handleCalculation(ctx context.Context, sessionID, expr string, sender openacp.SessionEventSender) (*openacp.PromptResponse, error) {
	_ = sender.SendAgentMessage(fmt.Sprintf("Let me calculate %s...\n", expr))

	_ = sender.SendToolCall(openacp.ToolCallUpdate{
		ToolCallID: fmt.Sprintf("calc_%d", time.Now().UnixNano()),
		Title:      "calculate",
		Status:     "in_progress",
		RawInput:   map[string]any{"expression": expr},
	})

	result, calcErr := evaluate(expr)
	status := "completed"
	rawOutput := map[string]any{"result": result}
	if calcErr != nil {
		status = "failed"
		rawOutput = map[string]any{"error": calcErr.Error()}
	}
	_ = sender.SendToolCall(openacp.ToolCallUpdate{
		ToolCallID: fmt.Sprintf("calc_%d", time.Now().UnixNano()),
		Title:      "calculate",
		Status:     status,
		RawOutput:  rawOutput,
	})

	if calcErr != nil {
		_ = sender.SendAgentMessage(fmt.Sprintf("Error: %s\n", calcErr))
	} else {
		_ = sender.SendAgentMessage(fmt.Sprintf("Result: %s = %s\n", expr, result))
	}
	return &openacp.PromptResponse{StopReason: openacp.StopReasonEndTurn}, nil
}

func (s *calcServer) OnCancel(ctx context.Context, sid openacp.SessionId) error {
	log.Printf("cancel: %s", sid)
	return nil
}

func (s *calcServer) OnLogout(ctx context.Context, req openacp.LogoutRequest) (*openacp.LogoutResponse, error) {
	return &openacp.LogoutResponse{}, nil
}

// ── Calculator ──

func extractExpression(input string) (string, bool) {
	lower := strings.ToLower(input)
	for _, prefix := range []string{"calculate ", "compute ", "eval ", "what is ", "what's "} {
		if after, ok := strings.CutPrefix(lower, prefix); ok {
			return strings.TrimSpace(after), true
		}
	}
	for _, op := range []string{" + ", " - ", " * ", " / "} {
		if strings.Contains(input, op) {
			return strings.TrimSpace(input), true
		}
	}
	return "", false
}

func evaluate(expr string) (string, error) {
	expr = strings.TrimSpace(expr)
	for _, op := range []string{"+", "-", "*", "/"} {
		idx := strings.Index(expr, " "+op+" ")
		if idx < 0 {
			continue
		}
		aStr := strings.TrimSpace(expr[:idx])
		bStr := strings.TrimSpace(expr[idx+len(" "+op+" "):])
		a, err1 := strconv.ParseFloat(aStr, 64)
		b, err2 := strconv.ParseFloat(bStr, 64)
		if err1 != nil || err2 != nil {
			return "", fmt.Errorf("invalid numbers in: %s", expr)
		}
		switch op {
		case "+":
			return formatFloat(a + b), nil
		case "-":
			return formatFloat(a - b), nil
		case "*":
			return formatFloat(a * b), nil
		case "/":
			if b == 0 {
				return "", fmt.Errorf("division by zero")
			}
			return formatFloat(a / b), nil
		}
	}
	return "", fmt.Errorf("no operator found in: %s", expr)
}

func formatFloat(f float64) string {
	if f == math.Trunc(f) {
		return fmt.Sprintf("%.0f", f)
	}
	return strconv.FormatFloat(f, 'g', -1, 64)
}

func firstLine(s string, maxLen int) string {
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		s = s[:idx]
	}
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
