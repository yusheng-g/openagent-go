package fs

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	openagent "github.com/yusheng-g/openagent-go"
)

// writeSkill creates <root>/<name>/SKILL.md with the given name and
// description. The directory tree is created on demand.
func writeSkill(t *testing.T, root, name, desc string) {
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

func namesOf(skills []openagent.SkillInfo) []string {
	out := make([]string, len(skills))
	for i, s := range skills {
		out[i] = s.Name
	}
	return out
}

// TestLoader_SingleRoot mirrors the classic single-tree behaviour.
func TestLoader_SingleRoot(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "alpha", "first")
	writeSkill(t, root, "beta", "second")

	skills, err := New(root).Discover(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got := namesOf(skills); len(got) != 2 {
		t.Errorf("got %d skills, want 2: %v", len(got), got)
	}
}

// TestLoader_NoRoots ensures an empty roots list yields no skills and no error.
func TestLoader_NoRoots(t *testing.T) {
	skills, err := New().Discover(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(skills) != 0 {
		t.Errorf("got %d skills, want 0", len(skills))
	}
}

// TestLoader_UnreadableRootSkipped verifies a missing root contributes
// nothing instead of failing the whole Discover.
func TestLoader_UnreadableRootSkipped(t *testing.T) {
	good := t.TempDir()
	writeSkill(t, good, "alpha", "first")
	missing := filepath.Join(t.TempDir(), "does-not-exist")

	skills, err := New(missing, good).Discover(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got := namesOf(skills); len(got) != 1 || got[0] != "alpha" {
		t.Errorf("got %v, want [alpha]", got)
	}
}

// TestLoader_OverrideSameName is the core guarantee: a skill in a later
// root overrides the same-name skill from an earlier root, keeping the
// original position and pointing Path at the later root's copy.
func TestLoader_OverrideSameName(t *testing.T) {
	home := t.TempDir() // earlier, less authoritative
	cwd := t.TempDir()  // later, more authoritative
	writeSkill(t, home, "shared", "from home")
	writeSkill(t, cwd, "shared", "from cwd")
	writeSkill(t, home, "only-home", "home exclusive")

	skills, err := New(home, cwd).Discover(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(skills) != 2 {
		t.Fatalf("got %d skills, want 2: %+v", len(skills), skills)
	}

	// Locate "shared" by name; its position is whatever ReadDir gave it
	// on first discovery — what matters is that the entry was overridden
	// in place (not appended as a duplicate).
	var shared *openagent.SkillInfo
	sharedCount := 0
	for i := range skills {
		if skills[i].Name == "shared" {
			shared = &skills[i]
			sharedCount++
		}
	}
	if sharedCount != 1 {
		t.Fatalf("found %d entries named shared, want 1 (override must not duplicate)", sharedCount)
	}
	// Content: overridden by cwd's copy.
	if shared.Description != "from cwd" {
		t.Errorf("shared description = %q, want %q", shared.Description, "from cwd")
	}
	// Path must point into cwd, the overriding root.
	if shared.Path != filepath.Join(cwd, "shared") {
		t.Errorf("shared Path = %q, want %q", shared.Path, filepath.Join(cwd, "shared"))
	}

	// only-home must still be present, untouched.
	foundOnlyHome := false
	for _, s := range skills {
		if s.Name == "only-home" {
			foundOnlyHome = true
			if s.Description != "home exclusive" {
				t.Errorf("only-home description = %q, want %q", s.Description, "home exclusive")
			}
		}
	}
	if !foundOnlyHome {
		t.Error("only-home missing from result")
	}
}

// TestLoader_DisjointNames keeps both skills when names differ.
func TestLoader_DisjointNames(t *testing.T) {
	home := t.TempDir()
	cwd := t.TempDir()
	writeSkill(t, home, "home-skill", "home only")
	writeSkill(t, cwd, "cwd-skill", "cwd only")

	skills, err := New(home, cwd).Discover(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(skills) != 2 {
		t.Fatalf("got %d skills, want 2: %+v", len(skills), skills)
	}
	got := map[string]bool{skills[0].Name: true, skills[1].Name: true}
	if !got["home-skill"] || !got["cwd-skill"] {
		t.Errorf("names = %v, want both home-skill and cwd-skill", skills)
	}
}

// TestLoader_OnlyHomeRoot verifies a single existing home root works on
// its own (cwd root absent).
func TestLoader_OnlyHomeRoot(t *testing.T) {
	home := t.TempDir()
	writeSkill(t, home, "alpha", "first")
	cwdMissing := filepath.Join(t.TempDir(), "nope")

	skills, err := New(home, cwdMissing).Discover(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got := namesOf(skills); len(got) != 1 || got[0] != "alpha" {
		t.Errorf("got %v, want [alpha]", got)
	}
}

// TestLoader_OnlyCwdRoot verifies a single existing cwd root works on
// its own (home root absent).
func TestLoader_OnlyCwdRoot(t *testing.T) {
	homeMissing := filepath.Join(t.TempDir(), "nope")
	cwd := t.TempDir()
	writeSkill(t, cwd, "beta", "second")

	skills, err := New(homeMissing, cwd).Discover(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got := namesOf(skills); len(got) != 1 || got[0] != "beta" {
		t.Errorf("got %v, want [beta]", got)
	}
}
