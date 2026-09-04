package acp

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"testing"
	"time"
)

// TestListConfigOptionsWireRoundtrip drives a real JSON-RPC
// session/list_config_options request through RunTransport and checks the
// response reaches the client (dispatch by method name + response
// serialization).
func TestListConfigOptionsWireRoundtrip(t *testing.T) {
	srv := NewServer("wire-test", "1.0.0", &floodHandler{})
	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	go func() { _ = srv.RunTransport(ctx, stdoutW, stdinR) }()

	go func() {
		defer stdinW.Close()
		req := `{"jsonrpc":"2.0","id":"1","method":"session/list_config_options","params":{}}` + "\n"
		_, _ = io.WriteString(stdinW, req)
	}()

	var got map[string]any
	sc := bufio.NewScanner(stdoutR)
	for sc.Scan() {
		var m map[string]any
		if err := json.Unmarshal(sc.Bytes(), &m); err == nil && m["id"] != nil {
			got = m
			break
		}
	}
	if got == nil {
		t.Fatal("no response received")
	}
	if got["error"] != nil {
		t.Fatalf("unexpected error: %v", got["error"])
	}
	if _, ok := got["result"].(map[string]any); !ok {
		t.Fatalf("result = %#v, want object", got["result"])
	}
}
