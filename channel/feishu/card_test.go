package feishu

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/yusheng-g/openagent-go/channel"
)

func TestBuildCardBasic(t *testing.T) {
	c := &channel.Card{
		Header:  channel.CardHeader{Title: "Hello"},
		Content: "**bold** text",
		Color:   channel.CardColorBlue,
	}
	result, err := BuildCard(c)
	if err != nil {
		t.Fatal(err)
	}

	var m map[string]any
	if err := json.Unmarshal([]byte(result), &m); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}

	assertString(t, m, "header.title.content", "Hello")
	assertString(t, m, "header.template", "blue")
	assertString(t, m, "schema", "2.0")

	// Schema 2.0 — no config/wide_screen_mode.
	if _, ok := m["config"]; ok {
		t.Error("schema 2.0 should not have config")
	}

	// body.elements[0] is the markdown element.
	body, _ := m["body"].(map[string]any)
	elems, _ := body["elements"].([]any)
	first, _ := elems[0].(map[string]any)
	if first["tag"] != "markdown" {
		t.Errorf("first element tag = %v, want markdown", first["tag"])
	}
	if first["content"] != "**bold** text" {
		t.Errorf("first element content = %v, want **bold** text", first["content"])
	}
}

func TestBuildCardEmptyContent(t *testing.T) {
	c := &channel.Card{
		Header: channel.CardHeader{Title: "X"},
		Color:  channel.CardColorGrey,
	}
	result, err := BuildCard(c)
	if err != nil {
		t.Fatal(err)
	}
	// Empty content should produce a placeholder element (hr), not "(empty)".
	if strings.Contains(result, "(empty)") {
		t.Errorf("empty content should not produce (empty), got: %s", result)
	}
	if !strings.Contains(result, `"tag":"hr"`) {
		t.Errorf("empty content should produce an hr placeholder, got: %s", result)
	}
}

func TestBuildCardWithFooter(t *testing.T) {
	c := &channel.Card{
		Header:  channel.CardHeader{Title: "X"},
		Content: "body",
		Footer:  "note text",
		Color:   channel.CardColorGreen,
	}
	result, err := BuildCard(c)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result, `"tag":"hr"`) {
		t.Error("footer should produce hr element")
	}
	if !strings.Contains(result, `"tag":"note"`) {
		t.Error("footer should produce note element")
	}
	if !strings.Contains(result, "note text") {
		t.Error("footer should contain note text")
	}
}

func TestBuildCardSubtitle(t *testing.T) {
	c := &channel.Card{
		Header:  channel.CardHeader{Title: "X", Subtitle: "sub"},
		Content: "body",
		Color:   channel.CardColorRed,
	}
	result, err := BuildCard(c)
	if err != nil {
		t.Fatal(err)
	}
	// JSON key is "subtitle", value should be "sub".
	assertJSONContains(t, result, `"subtitle"`)
	assertJSONContains(t, result, `"sub"`)
}

func TestBuildCardAllColors(t *testing.T) {
	colors := map[channel.CardColor]string{
		channel.CardColorBlue:   "blue",
		channel.CardColorGreen:  "green",
		channel.CardColorRed:    "red",
		channel.CardColorYellow: "yellow",
		channel.CardColorOrange: "orange",
		channel.CardColorPurple: "purple",
		channel.CardColorGrey:   "grey",
	}
	for col, expected := range colors {
		c := &channel.Card{Header: channel.CardHeader{Title: "X"}, Content: "x", Color: col}
		result, err := BuildCard(c)
		if err != nil {
			t.Fatalf("color %s failed: %v", col, err)
		}
		if !strings.Contains(result, `"template":"`+expected+`"`) {
			t.Errorf("color %s: expected template %q in result", col, expected)
		}
	}
}

func TestBuildCardNil(t *testing.T) {
	_, err := BuildCard(nil)
	if err == nil {
		t.Fatal("expected error for nil card")
	}
}

func TestBuildCardWithApproval(t *testing.T) {
	c := &channel.Card{
		Header:  channel.CardHeader{Title: "🔧 调用工具中"},
		Content: "running...",
		Color:   channel.CardColorYellow,
		Approval: &channel.CardApproval{
			ToolName:   "shell",
			Args:       `{"command":"rm -rf /tmp"}`,
			ApprovalID: "abc123",
		},
	}
	result, err := BuildCard(c)
	if err != nil {
		t.Fatal(err)
	}

	var m map[string]any
	if err := json.Unmarshal([]byte(result), &m); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// config.update_multi should be set when approval is present.
	cfg, _ := m["config"].(map[string]any)
	if cfg == nil {
		t.Fatal("expected config when approval is set")
	}
	if cfg["update_multi"] != true {
		t.Error("expected update_multi=true")
	}

	body, _ := m["body"].(map[string]any)
	elems, _ := body["elements"].([]any)

	// Last three elements: hr, markdown (approval context), column_set (3 buttons).
	n := len(elems)
	if n < 3 {
		t.Fatalf("expected at least 3 elements, got %d", n)
	}
	hr, _ := elems[n-3].(map[string]any)
	if hr["tag"] != "hr" {
		t.Errorf("expected hr before approval, got %v", hr["tag"])
	}
	md, _ := elems[n-2].(map[string]any)
	if md["tag"] != "markdown" {
		t.Errorf("expected markdown for approval context, got %v", md["tag"])
	}
	mdContent, _ := md["content"].(string)
	if !strings.Contains(mdContent, "shell") {
		t.Error("approval markdown should contain tool name")
	}
	if !strings.Contains(mdContent, "rm -rf /tmp") {
		t.Error("approval markdown should contain args")
	}
	colSet, _ := elems[n-1].(map[string]any)
	if colSet["tag"] != "column_set" {
		t.Errorf("expected column_set for buttons, got %v", colSet["tag"])
	}

	// Approval ID embedded in all 3 button values.
	count := strings.Count(result, `"approval_id":"abc123"`)
	if count != 3 {
		t.Errorf("expected 3 buttons with approval_id, got %d", count)
	}
}

func TestBuildCardWithoutApprovalHasNoConfig(t *testing.T) {
	c := &channel.Card{
		Header:  channel.CardHeader{Title: "X"},
		Content: "body",
		Color:   channel.CardColorBlue,
	}
	result, err := BuildCard(c)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	json.Unmarshal([]byte(result), &m)
	if _, ok := m["config"]; ok {
		t.Error("card without approval should not have config")
	}
}

func TestBuildCardTrimsContent(t *testing.T) {
	c := &channel.Card{
		Header:  channel.CardHeader{Title: "X"},
		Content: "  \n\n  hello  \n  ",
		Color:   channel.CardColorBlue,
	}
	result, err := BuildCard(c)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONContains(t, result, `"content":"hello"`)
}

// ── helpers ──

func assertString(t *testing.T, m map[string]any, path, want string) {
	t.Helper()
	parts := strings.Split(path, ".")
	cur := any(m)
	for i, p := range parts {
		mp, ok := cur.(map[string]any)
		if !ok {
			t.Fatalf("at %q: expected map, got %T", strings.Join(parts[:i], "."), cur)
		}
		v, ok := mp[p]
		if !ok {
			t.Fatalf("path %q not found", path)
		}
		if i == len(parts)-1 {
			if s, ok := v.(string); !ok || s != want {
				t.Errorf("%s = %q, want %q", path, v, want)
			}
			return
		}
		cur = v
	}
}

func assertJSONContains(t testing.TB, jsonStr, substr string) {
	t.Helper()
	if !strings.Contains(jsonStr, substr) {
		t.Errorf("expected JSON to contain %q", substr)
	}
}
