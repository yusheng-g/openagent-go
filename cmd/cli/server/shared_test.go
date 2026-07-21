package server

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	openagent "github.com/yusheng-g/openagent-go"
)

func TestSkillDirs_Priority(t *testing.T) {
	dirs := skillDirs()
	if len(dirs) < 1 {
		t.Fatal("skillDirs returned empty list")
	}
	// First entry should be project-level (cwd-based).
	if !strings.Contains(dirs[0], ".openagent") || !strings.Contains(dirs[0], "skills") {
		t.Errorf("unexpected project dir: %q", dirs[0])
	}
	// Second, if present, should be user-level (home-based).
	if len(dirs) >= 2 {
		if !strings.Contains(dirs[1], ".openagent") || !strings.Contains(dirs[1], "skills") {
			t.Errorf("unexpected home dir: %q", dirs[1])
		}
	}
}

func TestOpenSkillLoader_NoDirs(t *testing.T) {
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", "/nonexistent-path-for-test")
	defer os.Setenv("HOME", origHome)

	origWd, _ := os.Getwd()
	tmp := t.TempDir()
	os.Chdir(tmp)
	defer os.Chdir(origWd)

	sl := openSkillLoader()
	if sl != nil {
		t.Error("openSkillLoader should return nil when no .openagent/skills exists")
	}
}

func TestOpenSkillLoader_FindsDir(t *testing.T) {
	origWd, _ := os.Getwd()
	tmp := t.TempDir()
	os.Chdir(tmp)
	defer os.Chdir(origWd)

	skillsDir := filepath.Join(tmp, ".openagent", "skills")
	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	sl := openSkillLoader()
	if sl == nil {
		t.Fatal("openSkillLoader should find .openagent/skills directory")
	}
}

func TestBuildGuard_NilModel(t *testing.T) {
	// buildGuard with nil model: Guard holds a nil model, which will fail
	// at Check time (not construction time). Verify no panic.
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("buildGuard(nil) panicked: %v", r)
		}
	}()
	g := buildGuard(nil)
	if g == nil {
		t.Fatal("buildGuard returned nil")
	}
	// Output should also be non-nil.
	if o := g.Output(); o == nil {
		t.Fatal("guard.Output() returned nil")
	}
}

func TestBuildSlogHooks(t *testing.T) {
	hooks := buildSlogHooks()
	// sloghooks.New creates a valid hooks instance (may be wrapper around logger).
	if hooks == nil {
		t.Log("buildSlogHooks returned nil — verify if intentional")
	}
}

func TestBuildSlogObserver(t *testing.T) {
	obs := buildSlogObserver()
	if obs == nil {
		t.Fatal("buildSlogObserver returned nil")
	}
	// Calling ObserveStage should not panic.
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("ObserveStage panicked: %v", r)
		}
	}()
	obs.ObserveStage(context.Background(), openagent.StageEvent{
		Name:     "test-stage",
		Phase:    "start",
		Duration: 0,
	})
}
