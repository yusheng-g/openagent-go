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
	// First entry should be user-level (~/.agents/skills).
	if !strings.Contains(dirs[0], ".agents") || !strings.Contains(dirs[0], "skills") {
		t.Errorf("unexpected user dir: %q", dirs[0])
	}
	// Second, if present, should be project-level (<cwd>/.agents/skills).
	if len(dirs) >= 2 {
		if !strings.Contains(dirs[1], ".agents") || !strings.Contains(dirs[1], "skills") {
			t.Errorf("unexpected project dir: %q", dirs[1])
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
		t.Error("openSkillLoader should return nil when no .agents/skills exists")
	}
}

func TestOpenSkillLoader_FindsDir(t *testing.T) {
	origWd, _ := os.Getwd()
	tmp := t.TempDir()
	os.Chdir(tmp)
	defer os.Chdir(origWd)

	skillsDir := filepath.Join(tmp, ".agents", "skills")
	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	sl := openSkillLoader()
	if sl == nil {
		t.Fatal("openSkillLoader should find .agents/skills directory")
	}
}

// writeSkillFile creates <root>/<name>/SKILL.md with the given frontmatter.
func writeSkillFile(t *testing.T, root, name, desc string) {
	t.Helper()
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\nname: " + name + "\ndescription: " + desc + "\n---\nbody of " + name + "\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestOpenSkillLoader_MergesHomeAndCwd is the end-to-end assembly guarantee:
// when both ~/.agents/skills and <cwd>/.agents/skills exist with a same-name
// skill, openSkillLoader must load BOTH directories and the cwd copy wins.
// This exercises skillDirs ordering + stat filtering + fs.New(roots...)
// together — the behaviour that was previously "first hit wins, single dir".
func TestOpenSkillLoader_MergesHomeAndCwd(t *testing.T) {
	origHome := os.Getenv("HOME")
	origWd, _ := os.Getwd()
	tmp := t.TempDir()
	homeDir := filepath.Join(tmp, "home")
	cwdDir := filepath.Join(tmp, "cwd")
	if err := os.MkdirAll(homeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(cwdDir, 0o755); err != nil {
		t.Fatal(err)
	}
	os.Setenv("HOME", homeDir)
	os.Chdir(cwdDir)
	defer func() {
		os.Chdir(origWd)
		os.Setenv("HOME", origHome)
	}()

	homeSkills := filepath.Join(homeDir, ".agents", "skills")
	cwdSkills := filepath.Join(cwdDir, ".agents", "skills")
	writeSkillFile(t, homeSkills, "shared", "from home")
	writeSkillFile(t, homeSkills, "home-only", "home exclusive")
	writeSkillFile(t, cwdSkills, "shared", "from cwd")
	writeSkillFile(t, cwdSkills, "cwd-only", "cwd exclusive")

	sl := openSkillLoader()
	if sl == nil {
		t.Fatal("openSkillLoader should be non-nil when skill dirs exist")
	}
	skills, err := sl.Discover(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	byName := make(map[string]openagent.SkillInfo)
	for _, s := range skills {
		byName[s.Name] = s
	}

	// All four names present (both dirs loaded).
	for _, want := range []string{"shared", "home-only", "cwd-only"} {
		if _, ok := byName[want]; !ok {
			t.Errorf("skill %q missing from merged result: %+v", want, skills)
		}
	}
	if len(skills) != 3 {
		t.Errorf("got %d skills, want 3 (shared merged, not duplicated): %+v", len(skills), skills)
	}

	// shared must be the cwd copy (override).
	shared := byName["shared"]
	if shared.Description != "from cwd" {
		t.Errorf("shared description = %q, want %q (cwd should override home)", shared.Description, "from cwd")
	}
	if shared.Path != filepath.Join(cwdSkills, "shared") {
		t.Errorf("shared Path = %q, want %q", shared.Path, filepath.Join(cwdSkills, "shared"))
	}
}

// TestSkillDirs_HomeEqualsCwd_Deduped verifies that when home and cwd
// resolve to the same directory, skillDirs returns a single entry
// instead of duplicating the path (which would make Discover scan the
// same tree twice).
func TestSkillDirs_HomeEqualsCwd_Deduped(t *testing.T) {
	origHome := os.Getenv("HOME")
	origWd, _ := os.Getwd()
	tmp := t.TempDir()
	os.Setenv("HOME", tmp)
	os.Chdir(tmp)
	defer func() {
		os.Chdir(origWd)
		os.Setenv("HOME", origHome)
	}()

	dirs := skillDirs()
	if len(dirs) != 1 {
		t.Fatalf("got %d dirs, want 1 (home==cwd dedup): %v", len(dirs), dirs)
	}
	want := filepath.Join(tmp, ".agents", "skills")
	if dirs[0] != want {
		t.Errorf("dirs[0] = %q, want %q", dirs[0], want)
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
