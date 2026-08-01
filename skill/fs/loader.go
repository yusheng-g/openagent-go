// Package fs implements SkillLoader backed by a directory tree of SKILL.md files.
//
// Directory layout:
//
//	<root>/
//	  example-skill/
//	    SKILL.md
//	    scripts/...
//	  another-skill/
//	    SKILL.md
//
// Each SKILL.md begins with YAML frontmatter (--- ... ---) containing at minimum
// name and description. All frontmatter fields are preserved in Frontmatter.
// The body (after the closing ---) is loaded on demand via Load().
package fs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	openagent "github.com/yusheng-g/openagent-go"
)

// Loader discovers and loads skills from one or more directory trees.
// When multiple roots are given, Discover scans them in order and skills
// in later roots override same-name skills from earlier roots (the
// earlier entry keeps its position in the result, only its content is
// replaced). This lets a project-level root (<cwd>/.agents/skills) take
// priority over a user-level root (~/.agents/skills) when the project
// root is passed last.
type Loader struct {
	roots []string
}

// New creates a Loader rooted at the given directories. With a single
// root it behaves as a classic single-tree loader; passing multiple
// roots enables the override semantics described on Loader.
func New(roots ...string) *Loader {
	return &Loader{roots: roots}
}

// Discover scans each root for subdirectories containing SKILL.md, reads
// each file's YAML frontmatter, and returns a SkillInfo for each valid
// skill. Roots are processed in order; a skill from a later root with
// the same name as one from an earlier root replaces the earlier entry
// in place (preserving its position), while skills with new names are
// appended. Skills missing name or description are skipped. A root that
// cannot be read is treated as empty rather than failing the whole call.
func (l *Loader) Discover(ctx context.Context) ([]openagent.SkillInfo, error) {
	var skills []openagent.SkillInfo
	indexByName := make(map[string]int)

	for _, root := range l.roots {
		entries, err := os.ReadDir(root)
		if err != nil {
			// Missing/unreadable root contributes nothing.
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			skillDir := filepath.Join(root, entry.Name())
			mdPath := filepath.Join(skillDir, "SKILL.md")

			fm, body, err := parseFrontmatter(mdPath)
			if err != nil {
				continue
			}

			name, _ := fm["name"].(string)
			desc, _ := fm["description"].(string)
			if name == "" || desc == "" {
				continue
			}

			info := openagent.SkillInfo{
				Name:        name,
				Description: desc,
				Frontmatter: fm,
				Path:        skillDir,
			}
			_ = body

			if idx, ok := indexByName[name]; ok {
				skills[idx] = info // override in place, keep position
			} else {
				indexByName[name] = len(skills)
				skills = append(skills, info)
			}
		}
	}

	return skills, nil
}

// Load reads the SKILL.md for the given skill and returns the body
// (content after the closing YAML frontmatter).
func (l *Loader) Load(ctx context.Context, skill openagent.SkillInfo) (string, error) {
	mdPath := filepath.Join(skill.Path, "SKILL.md")
	_, body, err := parseFrontmatter(mdPath)
	return body, err
}

// parseFrontmatter splits a markdown file into YAML frontmatter (map) and body.
func parseFrontmatter(path string) (map[string]any, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}

	text := string(data)
	if !strings.HasPrefix(text, "---\n") {
		return nil, "", fmt.Errorf("no frontmatter")
	}

	// Find closing --- on its own line
	idx := strings.Index(text[4:], "\n---\n")
	if idx == -1 {
		// Try with trailing newline only
		if strings.HasSuffix(text[4:], "\n---") {
			idx = len(text[4:]) - 4
		} else {
			return nil, "", fmt.Errorf("unclosed frontmatter")
		}
	}

	yamlBlock := text[4 : 4+idx]
	body := text[4+idx+5:] // skip \n---\n

	var fm map[string]any
	if err := yaml.Unmarshal([]byte(yamlBlock), &fm); err != nil {
		return nil, "", fmt.Errorf("invalid YAML frontmatter: %w", err)
	}
	if fm == nil {
		fm = make(map[string]any)
	}

	return fm, body, nil
}
