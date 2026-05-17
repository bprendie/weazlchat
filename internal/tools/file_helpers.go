package tools

import (
	"os"
	"path/filepath"
	"strings"
)

func intParam(params map[string]any, key string, def, min, max int) int {
	if _, ok := params[key]; !ok {
		return def
	}
	n, err := getNumber(params, key)
	if err != nil {
		return def
	}
	v := int(n)
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func skipHidden(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." {
		return false
	}
	for _, part := range strings.Split(rel, string(os.PathSeparator)) {
		if strings.HasPrefix(part, ".") {
			return true
		}
	}
	return false
}

func matchPathGlob(pattern, rel string) bool {
	pattern = filepath.ToSlash(pattern)
	rel = filepath.ToSlash(rel)
	if strings.HasPrefix(pattern, "**/") {
		ok, _ := filepath.Match(strings.TrimPrefix(pattern, "**/"), filepath.Base(rel))
		return ok
	}
	ok, _ := filepath.Match(pattern, rel)
	return ok
}

func looksText(data []byte) bool {
	for i, b := range data {
		if i > 4096 {
			break
		}
		if b == 0 {
			return false
		}
	}
	return true
}
