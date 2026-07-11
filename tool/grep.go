package tool

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	openagent "github.com/yusheng-g/openagent-go"
)

// Grep searches for a pattern in workspace files.
type Grep struct {
	workDir string
}

func NewGrep(workDir string) *Grep {
	abs, _ := filepath.Abs(workDir)
	return &Grep{workDir: abs}
}

func (t *Grep) Definition() openagent.FunctionDefinition {
	return openagent.FunctionDefinition{
		Name: "grep",
		Description: "Search for a pattern in workspace files. " +
			"Returns matching file paths, line numbers, and content. " +
			"Use for finding usages, definitions, or patterns in the codebase.",
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"pattern": {
					"type": "string",
					"description": "Text or regex pattern to search for (case-sensitive unless (?i) flag used)"
				},
				"path": {
					"type": "string",
					"description": "Subdirectory to search (default: entire workspace)"
				},
				"glob": {
					"type": "string",
					"description": "File pattern filter, e.g., '*.go' or '*.{go,md}' (default: all text files)"
				}
			},
			"required": ["pattern"]
		}`),
	}
}

func (t *Grep) CanSelfApprove(args json.RawMessage) bool {
	var params struct {
		Path string `json:"path"`
	}
	json.Unmarshal(args, &params)
	if params.Path == "" {
		return true // default to workspace root is safe
	}
	abs, err := validatePath(t.workDir, params.Path)
	if err != nil {
		return false
	}
	return isWithinWorkspace(t.workDir, abs)
}

func (t *Grep) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		Pattern string `json:"pattern"`
		Path    string `json:"path"`
		Glob    string `json:"glob"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("grep: %w", err)
	}
	if params.Pattern == "" {
		return "", fmt.Errorf("grep: pattern is required")
	}

	searchDir, err := validatePath(t.workDir, params.Path)
	if err != nil {
		// Empty path defaults to workspace root.
		if params.Path == "" {
			searchDir = t.workDir
		} else {
			return "", err
		}
	}

	re, err := regexp.Compile(params.Pattern)
	if err != nil {
		return "", fmt.Errorf("grep: invalid pattern: %w", err)
	}

	const (
		maxMatches   = 200
		maxFileSize  = 1 * 1024 * 1024 // 1MB
		maxFiles     = 2000
	)

	var (
		matches    []string
		totalFiles int
		truncated  bool
	)

	err = filepath.WalkDir(searchDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil || ctx.Err() != nil {
			return nil // skip on error
		}
		if d.IsDir() {
			base := d.Name()
			if strings.HasPrefix(base, ".") && base != "." {
				return filepath.SkipDir // skip .git, .node_modules, etc.
			}
			return nil
		}

		// Glob filter.
		if params.Glob != "" {
			matched, _ := filepath.Match(params.Glob, d.Name())
			if !matched {
				return nil
			}
		}

		totalFiles++
		if totalFiles > maxFiles {
			truncated = true
			return filepath.SkipAll
		}

		info, err := d.Info()
		if err != nil || info.Size() > maxFileSize {
			return nil
		}

		rel, _ := filepath.Rel(t.workDir, path)

		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		scanner.Buffer(make([]byte, 64*1024), 1024*1024) // 1MB max line
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			line := scanner.Text()
			if re.MatchString(line) {
				if len(matches) >= maxMatches {
					truncated = true
					return filepath.SkipAll
				}
				matches = append(matches, fmt.Sprintf("%s:%d: %s", rel, lineNum, line))
			}
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("grep: %w", err)
	}

	if len(matches) == 0 {
		return "No matches found.", nil
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("%d matches:\n", len(matches)))
	for _, m := range matches {
		b.WriteString(m)
		b.WriteString("\n")
	}
	if truncated {
		b.WriteString("[results truncated]")
	}
	return b.String(), nil
}
