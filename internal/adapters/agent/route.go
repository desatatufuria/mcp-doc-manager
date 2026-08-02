package agent

import (
	"os"
	"path/filepath"
)

type route struct {
	root, path  string
	exists      bool
	prospective bool
}

func openCodeRoute(root string) (route, error) {
	prospective := root == ""
	if root == "" {
		config, err := os.UserConfigDir()
		if err != nil {
			return route{}, ErrUnsafeRoute
		}
		root = filepath.Join(config, "opencode")
	}
	if !filepath.IsAbs(root) || filepath.Clean(root) != root {
		return route{}, ErrUnsafeRoute
	}
	info, err := os.Lstat(root)
	if os.IsNotExist(err) && prospective {
		parent, parentErr := os.Lstat(filepath.Dir(root))
		if parentErr != nil || !parent.IsDir() || parent.Mode()&os.ModeSymlink != 0 {
			return route{}, ErrUnsafeRoute
		}
		return route{root: root, path: filepath.Join(root, "opencode.json"), prospective: true}, nil
	}
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return route{}, ErrUnsafeRoute
	}
	for _, name := range []string{"opencode.json", "opencode.jsonc"} {
		path := filepath.Join(root, name)
		info, err := os.Lstat(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return route{}, ErrUnsafeRoute
		}
		return route{root: root, path: path, exists: true}, nil
	}
	return route{root: root, path: filepath.Join(root, "opencode.json")}, nil
}
