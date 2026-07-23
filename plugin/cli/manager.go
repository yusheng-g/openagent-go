package plugin

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Manager loads WASM settings-injection plugins from directories or single files.
type Manager struct {
	plugins []string // paths from settings.json: dirs are scanned for *.wasm
}

// NewManager creates a Manager from plugin paths in settings.json.
// Each entry may be a directory (all *.wasm files inside are loaded
// in sorted order) or a single .wasm file.
func NewManager(paths []string) *Manager {
	return &Manager{plugins: paths}
}

// Pipe sends settings through every plugin in order. Scans directories
// for *.wasm files. loadFn receives file bytes, module name, and
// current settings, and must return the merged settings.
func (m *Manager) Pipe(ctx context.Context, initialSettings []byte, loadFn func(wasmBytes []byte, name string, settings []byte) ([]byte, string, string, error)) ([]byte, error) {
	settings := initialSettings
	for _, path := range m.plugins {
		files, err := m.ResolveWasmFiles(path)
		if err != nil {
			log.Printf("plugin: resolve %s: %v", path, err)
			continue
		}
		sort.Strings(files)
		for _, f := range files {
			name := filepath.Base(f)
			wasmBytes, err := os.ReadFile(f)
			if err != nil {
				log.Printf("plugin: read %s: %v", f, err)
				continue
			}
			merged, pluginName, pluginDesc, err := loadFn(wasmBytes, name, settings)
			if err != nil {
				return nil, fmt.Errorf("plugin %s: %w", name, err)
			}
			settings = merged
			log.Printf("plugin: loaded %s (%s)", pluginName, pluginDesc)
		}
	}
	return settings, nil
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

func (m *Manager) ResolveWasmFiles(path string) ([]string, error) {
	path = expandPath(path)
	if path == "" {
		return nil, nil
	}
	fi, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if fi.IsDir() {
		entries, err := os.ReadDir(path)
		if err != nil {
			return nil, err
		}
		var files []string
		for _, e := range entries {
			if !e.IsDir() && filepath.Ext(e.Name()) == ".wasm" {
				files = append(files, filepath.Join(path, e.Name()))
			}
		}
		return files, nil
	}
	return []string{path}, nil
}
