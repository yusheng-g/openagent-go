package kernel

import (
	"context"
	"encoding/json"
	"testing"

	openagent "github.com/yusheng-g/openagent-go"
	"github.com/yusheng-g/openagent-go/agent"
)

// nameTool is a minimal Tool identified only by its name, for testing
// tool-set composition without exercising real execution.
type nameTool string

func (n nameTool) Definition() openagent.FunctionDefinition {
	return openagent.FunctionDefinition{Name: string(n)}
}

func (n nameTool) Execute(_ context.Context, _ json.RawMessage) *openagent.ToolResult {
	return &openagent.ToolResult{Content: "ok"}
}

// TestNew_RegistersSubAgentsAsTools verifies that each configured SubAgent
// appears in the runtime's tool snapshot under its Name, so the model can call
// it. It also confirms sub_agent_send is registered alongside them.
func TestNew_RegistersSubAgentsAsTools(t *testing.T) {
	cfg := agent.New("test")
	cfg.SubAgents = []agent.SubAgent{
		{Name: "explorer", Tools: []string{"read", "ls", "grep", "shell"}, MaxTurns: 100},
		{Name: "researcher", Tools: []string{"websearch", "webfetch"}, MaxTurns: 100},
		{Name: "reviewer", Tools: []string{"read", "grep", "webfetch"}, MaxTurns: 100},
	}
	rt := New(cfg, Deps{
		Tools: []openagent.Tool{nameTool("read")},
	})

	names := toolNamesFromSnapshot(rt)
	for _, want := range []string{"explorer", "researcher", "reviewer"} {
		if !contains(names, want) {
			t.Errorf("%q sub-agent not registered as a tool; got %v", want, names)
		}
	}
	if !contains(names, "sub_agent_send") {
		t.Errorf("sub_agent_send should be registered alongside the sub-agents; got %v", names)
	}
}

// TestNew_NoSubAgentsOmitsSendTool verifies sub_agent_send is NOT registered
// when there are no sub-agents (no delegation targets → no follow-up surface).
func TestNew_NoSubAgentsOmitsSendTool(t *testing.T) {
	cfg := agent.New("test")
	rt := New(cfg, Deps{Tools: []openagent.Tool{nameTool("read")}})

	names := toolNamesFromSnapshot(rt)
	if contains(names, "sub_agent_send") {
		t.Errorf("sub_agent_send should not be registered without sub-agents; got %v", names)
	}
}

// TestFilterChildTools_ThreeSubAgents verifies the source-based split:
//   - explorer: local tools only (read/ls/grep/shell); web tools dropped.
//   - researcher: web tools + local context (read/ls/grep/shell); mutating dropped.
//   - reviewer: read/ls/grep/shell/webfetch (inspect, not collect); websearch dropped.
//
// All three drop mutating tools (write/edit) and sub-agent tools (no recursion).
func TestFilterChildTools_ThreeSubAgents(t *testing.T) {
	parent := []openagent.Tool{
		nameTool("read"), nameTool("write"), nameTool("edit"),
		nameTool("ls"), nameTool("grep"), nameTool("shell"),
		nameTool("websearch"), nameTool("webfetch"),
		nameTool("browser_navigate"), nameTool("browser_screenshot"),
		nameTool("browser_evaluate"),
		// Office tools: read-side (kept by read-only sub-agents) + write-side
		// (must be filtered out — sub-agents are read-only intent).
		nameTool("pptx_read"), nameTool("excel_read"), nameTool("word_read"),
		nameTool("pptx_write"), nameTool("excel_write"), nameTool("word_write"),
		nameTool("explorer"), nameTool("researcher"), nameTool("reviewer"),
	}
	subAgentNames := map[string]bool{"explorer": true, "researcher": true, "reviewer": true}

	officeRead := []string{"pptx_read", "excel_read", "word_read"}
	officeWrite := []string{"pptx_write", "excel_write", "word_write"}

	cases := []struct {
		name    string
		allow   []string
		keep    []string
		blocked []string
	}{
		{
			name:    "explorer",
			allow:   append([]string{"read", "ls", "grep", "shell"}, officeRead...),
			keep:    append([]string{"read", "ls", "grep", "shell"}, officeRead...),
			blocked: append([]string{"write", "edit", "websearch", "webfetch", "browser_navigate", "explorer", "researcher", "reviewer"}, officeWrite...),
		},
		{
			name:    "researcher",
			allow:   append([]string{"websearch", "webfetch", "browser_navigate", "browser_screenshot", "browser_evaluate", "read", "ls", "grep", "shell"}, officeRead...),
			keep:    append([]string{"websearch", "webfetch", "browser_navigate", "browser_screenshot", "browser_evaluate", "read", "ls", "grep", "shell"}, officeRead...),
			blocked: append([]string{"write", "edit", "explorer", "researcher", "reviewer"}, officeWrite...),
		},
		{
			name:    "reviewer",
			allow:   append([]string{"read", "ls", "grep", "shell", "webfetch"}, officeRead...),
			keep:    append([]string{"read", "ls", "grep", "shell", "webfetch"}, officeRead...),
			blocked: append([]string{"write", "edit", "websearch", "browser_navigate", "explorer", "researcher", "reviewer"}, officeWrite...),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := filterChildTools(parent, subAgentNames, nil, tc.allow)
			names := toolNamesFromFilter(got)
			for _, want := range tc.keep {
				if !contains(names, want) {
					t.Errorf("%s child should keep %q; got %v", tc.name, want, names)
				}
			}
			for _, blocked := range tc.blocked {
				if contains(names, blocked) {
					t.Errorf("%s child must not contain %q; got %v", tc.name, blocked, names)
				}
			}
		})
	}
}

// TestFilterChildTools_NilAllowlistInheritsAllButSubAgents verifies that a
// sub-agent with no tool allowlist inherits every parent tool except other
// sub-agent tools.
func TestFilterChildTools_NilAllowlistInheritsAllButSubAgents(t *testing.T) {
	parent := []openagent.Tool{
		nameTool("read"), nameTool("write"), nameTool("general"),
	}
	subAgentNames := map[string]bool{"general": true}

	got := filterChildTools(parent, subAgentNames, nil, nil)

	names := toolNamesFromFilter(got)
	if contains(names, "general") {
		t.Errorf("sub-agent must not see another sub-agent tool (no recursion); got %v", names)
	}
	if !contains(names, "read") || !contains(names, "write") {
		t.Errorf("nil allowlist should inherit read+write; got %v", names)
	}
}

// --- helpers ---

func toolNamesFromSnapshot(rt *Runtime) []string {
	tools := rt.SnapshotTools()
	out := make([]string, 0, len(tools))
	for _, tl := range tools {
		out = append(out, tl.Definition().Name)
	}
	return out
}

// toolNamesFromFilter extracts names from a filterChildTools result. The
// helper exists because filterChildTools returns []openagent.Tool.
func toolNamesFromFilter(tools []openagent.Tool) []string {
	out := make([]string, 0, len(tools))
	for _, tl := range tools {
		out = append(out, tl.Definition().Name)
	}
	return out
}

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
