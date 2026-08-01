package release

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ExtractTarGz writes only validated regular files to a new directory under root.
// It never replaces a caller-selected target or writes outside that new directory.
func ExtractTarGz(data []byte, root string, allowed map[string]struct{}, limit int64) (string, error) {
	info, err := os.Lstat(root)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", ErrUnsafeArchive
	}
	dir, err := os.MkdirTemp(root, ".docmanager-extract-")
	if err != nil {
		return "", ErrUnsafeArchive
	}
	fail := func(e error) (string, error) { os.RemoveAll(dir); return "", e }
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return fail(ErrUnsafeArchive)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	seen := map[string]bool{}
	var total int64
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fail(ErrUnsafeArchive)
		}
		name := filepath.Clean(h.Name)
		if (h.Typeflag != tar.TypeReg && h.Typeflag != tar.TypeRegA) || h.Size < 0 || filepath.IsAbs(h.Name) || name == "." || strings.HasPrefix(name, ".."+string(filepath.Separator)) || strings.Contains(name, string(filepath.Separator)) || seen[name] {
			return fail(ErrUnsafeArchive)
		}
		if _, ok := allowed[name]; !ok {
			return fail(ErrUnsafeArchive)
		}
		seen[name] = true
		if h.Size > limit-total {
			return fail(ErrDownloadBounds)
		}
		total += h.Size
		out, err := os.OpenFile(filepath.Join(dir, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o700)
		if err != nil {
			return fail(ErrUnsafeArchive)
		}
		n, err := io.Copy(out, io.LimitReader(tr, h.Size+1))
		closeErr := out.Close()
		if err != nil || closeErr != nil || n != h.Size {
			return fail(ErrUnsafeArchive)
		}
	}
	if len(seen) != len(allowed) {
		return fail(ErrUnsafeArchive)
	}
	return dir, nil
}
