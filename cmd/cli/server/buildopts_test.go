package server

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	openagent "github.com/yusheng-g/openagent-go"
)

// mockModel is a no-op Model used only to satisfy buildGuard's non-nil
// requirement. Its methods are never called by buildOpts (the Guard is
// constructed but not invoked), so they return zero values.
type mockModel struct{}

func (mockModel) ChatCompletion(ctx context.Context, req openagent.ChatCompletionRequest) (*openagent.ChatCompletionResponse, error) {
	return nil, nil
}
func (mockModel) ChatCompletionStream(ctx context.Context, req openagent.ChatCompletionRequest) (openagent.StreamReader, error) {
	return nil, nil
}
func (mockModel) ContextWindow() int { return 0 }

// chdirEmpty ensures openSkillLoader() finds no .openagent/skills by moving
// to a temp dir and pointing HOME elsewhere. Returns a restore func.
func chdirEmpty(t *testing.T) func() {
	t.Helper()
	origHome := os.Getenv("HOME")
	origWd, _ := os.Getwd()
	tmp := t.TempDir()
	os.Chdir(tmp)
	os.Setenv("HOME", filepath.Join(tmp, "no-home"))
	return func() {
		os.Chdir(origWd)
		os.Setenv("HOME", origHome)
	}
}

// chdirWithSkills creates .openagent/skills in a temp cwd so openSkillLoader
// returns a non-nil loader.
func chdirWithSkills(t *testing.T) func() {
	t.Helper()
	origHome := os.Getenv("HOME")
	origWd, _ := os.Getwd()
	tmp := t.TempDir()
	os.Chdir(tmp)
	os.Setenv("HOME", filepath.Join(tmp, "no-home"))
	if err := os.MkdirAll(filepath.Join(tmp, ".openagent", "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	return func() {
		os.Chdir(origWd)
		os.Setenv("HOME", origHome)
	}
}

// applyOpts runs buildOpts over a base option slice and constructs a real
// Agent, so we can inspect the resulting exported fields.
func applyOpts(caps Capabilities, model openagent.Model) *openagent.Agent {
	opts := buildOpts(nil, caps, model)
	return openagent.NewAgent("test", opts...)
}

func TestBuildOpts_AllOff(t *testing.T) {
	restore := chdirEmpty(t)
	defer restore()
	// Capabilities zero value: skills/guard/hooks/observer all default OFF.
	agent := applyOpts(Capabilities{}, mockModel{})
	if agent.SkillLoader != nil {
		t.Error("SkillLoader should be nil when OnSkills=false")
	}
	if agent.InGuard != nil {
		t.Error("InGuard should be nil when OnGuard=false")
	}
	if agent.OutGuard != nil {
		t.Error("OutGuard should be nil when OnGuard=false")
	}
	if agent.Hooks != nil {
		t.Error("Hooks should be nil when OnHooks=false")
	}
	if agent.Observer != nil {
		t.Error("Observer should be nil when OnObserver=false")
	}
}

func TestBuildOpts_HooksAndObserver(t *testing.T) {
	restore := chdirEmpty(t)
	defer restore()
	on := true
	caps := Capabilities{Hooks: &on, Observer: &on}
	agent := applyOpts(caps, mockModel{})
	if agent.Hooks == nil {
		t.Error("Hooks should be attached when OnHooks=true")
	}
	if agent.Observer == nil {
		t.Error("Observer should be attached when OnObserver=true")
	}
	// Guard/skills still off.
	if agent.InGuard != nil || agent.SkillLoader != nil {
		t.Error("Guard/Skills should remain nil")
	}
}

func TestBuildOpts_Guard_RequiresModel(t *testing.T) {
	restore := chdirEmpty(t)
	defer restore()
	on := true
	// OnGuard=true but model is nil → guard must be skipped (no judge model).
	agent := applyOpts(Capabilities{Guard: &on}, nil)
	if agent.InGuard != nil {
		t.Error("InGuard should be nil when model is nil (no judge)")
	}
	if agent.OutGuard != nil {
		t.Error("OutGuard should be nil when model is nil (no judge)")
	}
}

func TestBuildOpts_Guard_WithModel(t *testing.T) {
	restore := chdirEmpty(t)
	defer restore()
	on := true
	agent := applyOpts(Capabilities{Guard: &on}, mockModel{})
	if agent.InGuard == nil {
		t.Error("InGuard should be attached when OnGuard=true and model!=nil")
	}
	if agent.OutGuard == nil {
		t.Error("OutGuard should be attached when OnGuard=true and model!=nil")
	}
}

func TestBuildOpts_Skills_NoDir(t *testing.T) {
	restore := chdirEmpty(t)
	defer restore()
	on := true
	// OnSkills=true but no .openagent/skills dir → loader stays nil.
	agent := applyOpts(Capabilities{Skills: &on}, mockModel{})
	if agent.SkillLoader != nil {
		t.Error("SkillLoader should be nil when no skills dir exists")
	}
}

func TestBuildOpts_Skills_WithDir(t *testing.T) {
	restore := chdirWithSkills(t)
	defer restore()
	on := true
	agent := applyOpts(Capabilities{Skills: &on}, mockModel{})
	if agent.SkillLoader == nil {
		t.Error("SkillLoader should be attached when skills dir exists and OnSkills=true")
	}
}

func TestBuildOpts_AllOn(t *testing.T) {
	restore := chdirWithSkills(t)
	defer restore()
	on := true
	caps := Capabilities{Skills: &on, Guard: &on, Hooks: &on, Observer: &on}
	agent := applyOpts(caps, mockModel{})
	if agent.SkillLoader == nil {
		t.Error("SkillLoader should be attached")
	}
	if agent.InGuard == nil || agent.OutGuard == nil {
		t.Error("Guard should be attached")
	}
	if agent.Hooks == nil {
		t.Error("Hooks should be attached")
	}
	if agent.Observer == nil {
		t.Error("Observer should be attached")
	}
}

// TestBuildOpts_PreservesBaseOpts verifies buildOpts appends to (not
// replaces) the supplied option slice — callers rely on this to keep
// their base options (model, memory, tools, prompts, maxTurns).
func TestBuildOpts_PreservesBaseOpts(t *testing.T) {
	restore := chdirEmpty(t)
	defer restore()
	on := true
	base := []openagent.AgentOption{
		openagent.WithMaxTurns(42),
		openagent.WithSystemPrompts("hello"),
	}
	opts := buildOpts(base, Capabilities{Hooks: &on}, mockModel{})
	agent := openagent.NewAgent("t", opts...)
	if agent.MaxTurns != 42 {
		t.Errorf("MaxTurns = %d, want 42 (base option must survive)", agent.MaxTurns)
	}
	if len(agent.SystemPrompts) != 1 || agent.SystemPrompts[0] != "hello" {
		t.Errorf("SystemPrompts = %v, want [hello]", agent.SystemPrompts)
	}
	if agent.Hooks == nil {
		t.Error("Hooks should be appended by buildOpts")
	}
}
