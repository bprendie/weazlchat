package storage

import (
	"os"
	"path/filepath"
)

func mkdirFor(path string) error {
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}
	return os.MkdirAll(dir, 0o700)
}
