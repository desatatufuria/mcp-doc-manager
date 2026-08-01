package release

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type tarEntry struct {
	name, body string
	typ        byte
	link       string
}

func archive(t *testing.T, entries ...tarEntry) []byte {
	t.Helper()
	var raw bytes.Buffer
	gz := gzip.NewWriter(&raw)
	tw := tar.NewWriter(gz)
	for _, e := range entries {
		if err := tw.WriteHeader(&tar.Header{Name: e.name, Mode: 0755, Size: int64(len(e.body)), Typeflag: e.typ, Linkname: e.link}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(e.body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return raw.Bytes()
}

func TestExtractTarGz(t *testing.T) {
	root := t.TempDir()
	sentinel := filepath.Join(root, "target")
	if err := os.WriteFile(sentinel, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	allowed := map[string]struct{}{"docmanager": {}, "LICENSE": {}}
	for _, tc := range []struct {
		name string
		in   []byte
		want error
	}{
		{"traversal", archive(t, tarEntry{"../escape", "x", tar.TypeReg, ""}), ErrUnsafeArchive},
		{"absolute", archive(t, tarEntry{"/escape", "x", tar.TypeReg, ""}), ErrUnsafeArchive},
		{"symlink", archive(t, tarEntry{"docmanager", "", tar.TypeSymlink, "../x"}), ErrUnsafeArchive},
		{"hardlink", archive(t, tarEntry{"docmanager", "", tar.TypeLink, "x"}), ErrUnsafeArchive},
		{"duplicate", archive(t, tarEntry{"docmanager", "a", tar.TypeReg, ""}, tarEntry{"docmanager", "b", tar.TypeReg, ""}), ErrUnsafeArchive},
		{"unexpected", archive(t, tarEntry{"other", "x", tar.TypeReg, ""}), ErrUnsafeArchive},
		{"decompression-limit", archive(t, tarEntry{"docmanager", "four", tar.TypeReg, ""}), ErrDownloadBounds},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ExtractTarGz(tc.in, root, allowed, 3)
			if !errors.Is(err, tc.want) {
				t.Fatalf("err=%v want %v", err, tc.want)
			}
			got, _ := os.ReadFile(sentinel)
			if string(got) != "old" {
				t.Fatalf("target replaced: %q", got)
			}
		})
	}
	t.Run("extracts only exact regular authorized contents in isolation", func(t *testing.T) {
		dir, err := ExtractTarGz(archive(t, tarEntry{"docmanager", "bin", tar.TypeReg, ""}, tarEntry{"LICENSE", "ok", tar.TypeReg, ""}), root, allowed, 8)
		if err != nil {
			t.Fatal(err)
		}
		for name, want := range map[string]string{"docmanager": "bin", "LICENSE": "ok"} {
			b, err := os.ReadFile(filepath.Join(dir, name))
			if err != nil || string(b) != want {
				t.Fatalf("%s=%q err=%v", name, b, err)
			}
		}
		if got, _ := os.ReadFile(sentinel); string(got) != "old" {
			t.Fatalf("target replaced: %q", got)
		}
	})
}
