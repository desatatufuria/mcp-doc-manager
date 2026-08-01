package agent

import (
	"os"
	"path/filepath"
)

func openCodeRoute(root string) (string, error) {
	if root == "" {
		root = filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "opencode")
	}
	if !filepath.IsAbs(root) || filepath.Clean(root) != root {
		return "", ErrUnsafeRoute
	}
	info, err := os.Lstat(root)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", ErrUnsafeRoute
	}
	for _, name := range []string{"opencode.json", "opencode.jsonc"} {
		path := filepath.Join(root, name)
		info, err := os.Lstat(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return "", ErrUnsafeRoute
		}
		return path, nil
	}
	return filepath.Join(root, "opencode.json"), nil
}
