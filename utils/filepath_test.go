package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizePath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	tests := []struct {
		name string
		in   string
		want string
	}{
		{"bare tilde", "~", home},
		{"tilde slash", "~/", home},
		{"tilde subpath", "~/projects/foo", filepath.Join(home, "projects/foo")},
		{"absolute unchanged", "/var/lib/data", "/var/lib/data"},
		{"relative unchanged", "relative/path", "relative/path"},
		{"empty falls back to home", "", home},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := NormalizePath(tc.in)
			if got != tc.want {
				t.Errorf("NormalizePath(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestNormalizePathHomeUnavailable(t *testing.T) {
	t.Setenv("HOME", "")
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}

	t.Run("tilde falls back to process cwd", func(t *testing.T) {
		got := NormalizePath("~")
		if got != wd {
			t.Errorf("NormalizePath(\"~\") with no HOME = %q, want %q", got, wd)
		}
	})

	t.Run("empty falls back to process cwd", func(t *testing.T) {
		got := NormalizePath("")
		if got != wd {
			t.Errorf("NormalizePath(\"\") with no HOME = %q, want %q", got, wd)
		}
	})
}
