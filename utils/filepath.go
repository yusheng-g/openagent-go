package utils

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// NormalizePath expands a leading "~" or "~/" in path to the user's home
// directory. If path is empty or the home directory cannot be resolved, it
// falls back to the process working directory so callers always receive a
// usable absolute path. Non-tilde, non-empty paths are returned unchanged.
func NormalizePath(path string) string {
	if path != "" && path != "~" && !strings.HasPrefix(path, "~/") {
		return path
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		switch {
		case path == "":
			slog.Warn("path is empty; falling back to home directory", "fallback", home)
			return home
		case path == "~":
			return home
		default:
			return filepath.Join(home, path[2:])
		}
	}
	if wd, err := os.Getwd(); err == nil {
		slog.Warn("path cannot be resolved; falling back to process cwd", "path", path, "fallback", wd)
		return wd
	}
	return path
}
