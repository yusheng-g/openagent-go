package huaweicloud

import (
	"embed"
	"io/fs"
)

// skillsFS holds the embedded skills directory. Each subdirectory is a
// skill (terraform-deployment, pricing, troubleshoot) containing SKILL.md
// and reference files (examples, guides). The server LLM loads these via
// the standard skill system and browses reference files with read/grep/ls.
//
//go:embed skills/*
var skillsFS embed.FS

// Skills returns the embedded skills directory as an fs.FS rooted at "skills/".
func Skills() fs.FS {
	sub, err := fs.Sub(skillsFS, "skills")
	if err != nil {
		return skillsFS
	}
	return sub
}
