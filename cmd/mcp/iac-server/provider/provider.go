// Package provider defines the CloudProvider interface for iac-server.
//
// A CloudProvider provides cloud identity, credentials, and an embedded
// skills directory (terraform deployment guide + examples, pricing guide,
// troubleshoot guide). The skills are compiled into the provider
// implementation via go:embed — iac-server extracts them to disk at
// startup and uses the standard skill/fs loader.
//
// Adding a cloud = implement this interface + embed a skills/ directory.
// Nothing in server core or iac/ changes.
package provider

import "io/fs"

// CloudProvider abstracts a cloud provider for iac-server.
//
// Implementations embed their skills directory at compile time. Each
// subdirectory of Skills() is a skill (with SKILL.md + reference files).
// iac-server extracts Skills() to disk at startup for the skill loader
// and standard read/grep/ls tools.
//
// Adding a cloud = implement this interface + embed skills. Nothing in
// server core or iac/ changes.
type CloudProvider interface {
	// Name returns the cloud identifier, e.g. "huaweicloud".
	Name() string

	// Env returns cloud credential environment variables for the
	// terraform subprocess. Reads from the process environment at
	// call time so secrets never persist in the struct.
	Env() map[string]string

	// Skills returns the embedded skills directory as an fs.FS.
	// Each subdirectory is a skill containing SKILL.md and optional
	// reference files (examples, guides, etc.).
	// nil means no skills are available for this cloud.
	Skills() fs.FS
}
