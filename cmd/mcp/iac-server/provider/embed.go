package provider

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// ExtractSkills extracts an embedded skills filesystem (fs.FS) to a
// destination directory on disk, overwriting any existing files. This
// is needed because the standard read/grep/ls tools and skill/fs loader
// operate on the OS filesystem, not embed.FS.
//
// Extraction runs unconditionally on every startup to ensure the disk
// copy always matches the embedded version. This is cheap — 1000+ small
// files write in well under a second.
func ExtractSkills(fsys fs.FS, dst string) error {
	if fsys == nil {
		return nil
	}

	return fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return os.MkdirAll(filepath.Join(dst, path), 0755)
		}

		target := filepath.Join(dst, path)

		srcFile, err := fsys.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		dstFile, err := os.Create(target)
		if err != nil {
			return err
		}
		defer dstFile.Close()

		_, err = io.Copy(dstFile, srcFile)
		return err
	})
}
