package prompt

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"testing"

	openagent "github.com/yusheng-g/openagent-go"
)

func TestDefaultCLI_Build(t *testing.T) {
	b := DefaultCLI("gpt-4o", 128000)
	pb := b.Build()

	input := openagent.PromptInput{
		ProjectContext: "Workspace: /home/test",
		WorkingMessages: []openagent.Message{
			openagent.UserMessage("hello"),
		},
	}

	msgs, err := pb(context.Background(), input)
	if err != nil {
		t.Fatalf("Build() returned error: %v", err)
	}
	if len(msgs) < 3 {
		t.Fatalf("expected at least 3 messages (static + dynamic + working), got %d", len(msgs))
	}

	sysMsgs := 0
	for _, m := range msgs {
		if m.Role == openagent.RoleSystem {
			sysMsgs++
		}
	}
	if sysMsgs < 2 {
		t.Fatalf("expected at least 2 system messages (static + dynamic), got %d", sysMsgs)
	}

	foundEnv := false
	for _, m := range msgs {
		if strings.Contains(m.Content, runtime.GOOS) && strings.Contains(m.Content, runtime.GOARCH) {
			foundEnv = true
			break
		}
	}
	if !foundEnv {
		t.Error("dynamic system message should contain OS and architecture")
	}

	lastMsg := msgs[len(msgs)-1]
	if lastMsg.Role != openagent.RoleUser || lastMsg.Content != "hello" {
		t.Errorf("last message should be the working user message, got role=%s content=%q", lastMsg.Role, lastMsg.Content)
	}
}

func TestDefaultServer_Build(t *testing.T) {
	b := DefaultServer("gpt-4o", 128000)
	pb := b.Build()

	input := openagent.PromptInput{
		WorkingMessages: []openagent.Message{
			openagent.UserMessage("hello"),
		},
	}

	msgs, err := pb(context.Background(), input)
	if err != nil {
		t.Fatalf("Build() returned error: %v", err)
	}
	if len(msgs) < 3 {
		t.Fatalf("expected at least 3 messages, got %d", len(msgs))
	}
}

func TestBuilder_SectionSorting(t *testing.T) {
	b := NewBuilder("gpt-4o", 128000)

	b.AddStatic(Section{Name: "C", Priority: 30, Static: true, Level: 2, Content: "c"})
	b.AddStatic(Section{Name: "A", Priority: 10, Static: true, Level: 2, Content: "a"})
	b.AddStatic(Section{Name: "B", Priority: 20, Static: true, Level: 2, Content: "b"})

	pb := b.Build()
	msgs, err := pb(context.Background(), openagent.PromptInput{})
	if err != nil {
		t.Fatalf("Build() returned error: %v", err)
	}
	if len(msgs) == 0 {
		t.Fatal("expected at least 1 system message")
	}

	idxA := strings.Index(msgs[0].Content, "## A")
	idxB := strings.Index(msgs[0].Content, "## B")
	idxC := strings.Index(msgs[0].Content, "## C")
	if idxA < 0 || idxB < 0 || idxC < 0 {
		t.Fatal("missing sections in output")
	}
	if !(idxA < idxB && idxB < idxC) {
		t.Error("sections not sorted by priority")
	}
}

func TestBuilder_SkipSection(t *testing.T) {
	b := NewBuilder("gpt-4o", 128000)

	skipCalled := false
	b.AddStatic(Section{
		Name:     "SkipMe",
		Priority: 10,
		Static:   true,
		Level:    2,
		Content:  "should not appear",
		Skip: func(input openagent.PromptInput) bool {
			skipCalled = true
			return true
		},
	})
	b.AddStatic(Section{
		Name:     "Visible",
		Priority: 20,
		Static:   true,
		Level:    2,
		Content:  "visible content",
	})

	pb := b.Build()
	msgs, err := pb(context.Background(), openagent.PromptInput{})
	if err != nil {
		t.Fatalf("Build() returned error: %v", err)
	}
	if !skipCalled {
		t.Error("Skip function was not called")
	}
	if strings.Contains(msgs[0].Content, "SkipMe") {
		t.Error("skipped section should not appear in output")
	}
	if !strings.Contains(msgs[0].Content, "Visible") {
		t.Error("visible section should appear in output")
	}
}

func TestBuilder_EmptySection(t *testing.T) {
	b := NewBuilder("gpt-4o", 128000)
	b.AddStatic(Section{Name: "Empty", Priority: 10, Static: true, Level: 2, Content: ""})

	pb := b.Build()
	msgs, err := pb(context.Background(), openagent.PromptInput{})
	if err != nil {
		t.Fatalf("Build() returned error: %v", err)
	}
	if len(msgs) != 0 {
		t.Error("empty section should not produce output")
	}
}

func TestBuilder_CompressedSummary(t *testing.T) {
	b := NewBuilder("gpt-4o", 128000)

	pb := b.Build()
	input := openagent.PromptInput{
		Compressed: &openagent.CompressedContext{
			Summary: "This is a summary of earlier conversation.",
			Hints: []openagent.RetrievalHint{
				{Description: "discussion about auth", Query: "auth"},
			},
		},
	}

	msgs, err := pb(context.Background(), input)
	if err != nil {
		t.Fatalf("Build() returned error: %v", err)
	}

	found := false
	for _, m := range msgs {
		if strings.Contains(m.Content, "Conversation Summary") &&
			strings.Contains(m.Content, "earlier conversation") &&
			strings.Contains(m.Content, "Retrieval Hints") {
			found = true
			break
		}
	}
	if !found {
		t.Error("compressed summary message not found in output")
	}
}

func TestBuilder_AvailableSkills(t *testing.T) {
	b := NewBuilder("gpt-4o", 128000)

	pb := b.Build()
	input := openagent.PromptInput{
		AvailableSkills: []openagent.SkillInfo{
			{Name: "web-dev", Frontmatter: map[string]any{"description": "web development helpers"}},
		},
	}

	msgs, err := pb(context.Background(), input)
	if err != nil {
		t.Fatalf("Build() returned error: %v", err)
	}

	found := false
	for _, m := range msgs {
		if strings.Contains(m.Content, "Available Skills") && strings.Contains(m.Content, "web-dev") {
			found = true
			break
		}
	}
	if !found {
		t.Error("available skills message not found in output")
	}
}

func TestBuilder_LoadedSkills(t *testing.T) {
	b := NewBuilder("gpt-4o", 128000)

	pb := b.Build()
	input := openagent.PromptInput{
		LoadedSkills: map[string]string{
			"my-skill": "skill body content",
		},
	}

	msgs, err := pb(context.Background(), input)
	if err != nil {
		t.Fatalf("Build() returned error: %v", err)
	}

	found := false
	for _, m := range msgs {
		if strings.Contains(m.Content, "Loaded Skill: my-skill") && strings.Contains(m.Content, "skill body content") {
			found = true
			break
		}
	}
	if !found {
		t.Error("loaded skill message not found in output")
	}
}

func TestBuilder_BudgetTrimsContent(t *testing.T) {
	b := NewBuilder("gpt-4o", 1000)

	longContent := strings.Repeat("very long text that will consume many tokens. ", 200)
	b.AddStatic(Section{Name: "Long", Priority: 10, Static: true, Level: 2, Content: longContent})

	pb := b.Build()
	msgs, err := pb(context.Background(), openagent.PromptInput{
		WorkingMessages: []openagent.Message{
			openagent.UserMessage("hello"),
		},
	})
	if err != nil {
		t.Fatalf("Build() returned error: %v", err)
	}

	staticTokens := b.countTokens(longContent)
	budget := int(float64(1000) * 0.95 * 0.12)

	if staticTokens <= budget {
		t.Skip("content doesn't exceed budget — trimming not triggered")
	}

	trimmedTokens := b.countTokens(msgs[0].Content)
	if trimmedTokens > budget {
		t.Errorf("trimmed content tokens (%d) should be <= budget (%d)", trimmedTokens, budget)
	}
	if !strings.Contains(msgs[0].Content, "[truncated]") {
		t.Error("trimmed content should contain [truncated] marker")
	}
}

func TestDefaultCLI_ReservedSectionsAreSkipped(t *testing.T) {
	b := DefaultCLI("gpt-4o", 128000)
	pb := b.Build()

	msgs, err := pb(context.Background(), openagent.PromptInput{})
	if err != nil {
		t.Fatalf("Build() returned error: %v", err)
	}

	for _, m := range msgs {
		if strings.Contains(m.Content, "Current Plan") ||
			strings.Contains(m.Content, "## Scratchpad") ||
			strings.Contains(m.Content, "Episodic Memory") ||
			strings.Contains(m.Content, "Semantic Memory") {
			t.Errorf("reserved section should not appear: %s", m.Content[:200])
		}
	}
}

func TestDefaultCLI_UserProfile(t *testing.T) {
	b := DefaultCLI("gpt-4o", 128000)
	pb := b.Build()

	msgs, err := pb(context.Background(), openagent.PromptInput{
		UserProfile: "Expert Go developer",
	})
	if err != nil {
		t.Fatalf("Build() returned error: %v", err)
	}

	found := false
	for _, m := range msgs {
		if strings.Contains(m.Content, "User Profile") && strings.Contains(m.Content, "Expert Go") {
			found = true
			break
		}
	}
	if !found {
		t.Error("user profile section not found in output")
	}
}

func TestDefaultCLI_AgentDescription(t *testing.T) {
	b := DefaultCLI("gpt-4o", 128000)
	pb := b.Build()

	msgs, err := pb(context.Background(), openagent.PromptInput{
		AgentDescription: "A specialized billing agent.",
	})
	if err != nil {
		t.Fatalf("Build() returned error: %v", err)
	}
	if len(msgs) < 2 {
		t.Fatal("expected at least 2 messages")
	}
}

func TestDynamicEnvironment_ContainsOSInfo(t *testing.T) {
	b := DefaultCLI("gpt-4o", 128000)
	pb := b.Build()

	msgs, err := pb(context.Background(), openagent.PromptInput{
		ProjectContext: "Workspace: /tmp/test",
	})
	if err != nil {
		t.Fatalf("Build() returned error: %v", err)
	}

	found := false
	for _, m := range msgs {
		if strings.Contains(m.Content, fmt.Sprintf("operating_system: %s", runtime.GOOS)) &&
			strings.Contains(m.Content, fmt.Sprintf("cpu_architecture: %s", runtime.GOARCH)) &&
			strings.Contains(m.Content, "working_directory: /tmp/test") {
			found = true
			break
		}
	}
	if !found {
		t.Error("environment section missing OS/arch/workspace info")
	}
}

func TestEmptyProjectContext(t *testing.T) {
	b := DefaultCLI("gpt-4o", 128000)
	pb := b.Build()

	msgs, err := pb(context.Background(), openagent.PromptInput{})
	if err != nil {
		t.Fatalf("Build() returned error: %v", err)
	}

	sysCount := 0
	for _, m := range msgs {
		if m.Role == openagent.RoleSystem {
			sysCount++
		}
	}
	if sysCount < 2 {
		t.Errorf("expected at least 2 system messages, got %d", sysCount)
	}
}
