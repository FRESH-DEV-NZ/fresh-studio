package promptloader

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Loader struct {
	root string
}

func New(root string) *Loader {
	return &Loader{
		root: root,
	}
}

func (l *Loader) Load(path string) (string, error) {
	if filepath.IsAbs(path) ||
		strings.Contains(path, "..") {
		return "", fmt.Errorf("invalid prompt path")
	}

	if filepath.Ext(path) != ".md" {
		return "", fmt.Errorf("prompt must be a Markdown file")
	}

	fullPath := filepath.Join(l.root, filepath.Clean(path))

	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("load prompt %q: %w", path, err)
	}

	return strings.TrimSpace(string(content)), nil
}
