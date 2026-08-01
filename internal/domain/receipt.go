package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
)

type DocumentationBlob struct {
	Path   string `json:"path"`
	Digest string `json:"digest"`
}

type ReceiptVersions struct {
	Schema string `json:"schema"`
	Tool   string `json:"tool"`
}

type Receipt struct {
	Digest        string              `json:"digest"`
	Scope         Scope               `json:"scope"`
	Evidence      string              `json:"evidence"`
	ChangedPaths  []string            `json:"changed_paths"`
	Documentation []DocumentationBlob `json:"documentation"`
	Outcome       Outcome             `json:"outcome"`
	Versions      ReceiptVersions     `json:"versions"`
}

func NewReceipt(report Report, documentation []DocumentationBlob, versions ReceiptVersions) (Receipt, error) {
	paths, err := canonicalPaths(report.Evidence.ChangedPaths)
	if err != nil {
		return Receipt{}, err
	}
	docs := append([]DocumentationBlob(nil), documentation...)
	sort.Slice(docs, func(i, j int) bool {
		if docs[i].Path == docs[j].Path {
			return docs[i].Digest < docs[j].Digest
		}
		return docs[i].Path < docs[j].Path
	})
	for i := 1; i < len(docs); i++ {
		if docs[i-1].Path == docs[i].Path {
			return Receipt{}, ErrReceiptMismatch
		}
	}
	receipt := Receipt{Scope: report.Evidence.Scope, Evidence: report.Evidence.Identity, ChangedPaths: paths, Documentation: docs, Outcome: report.Outcome, Versions: versions}
	canonical := receipt.canonical()
	digest := sha256.Sum256(canonical)
	receipt.Digest = "sha256:" + hex.EncodeToString(digest[:])
	return receipt, nil
}

func (r Receipt) ValidateDigest() error {
	paths, err := canonicalPaths(r.ChangedPaths)
	if err != nil {
		return ErrReceiptMismatch
	}
	if !sameStrings(paths, r.ChangedPaths) {
		return ErrReceiptMismatch
	}
	docs := append([]DocumentationBlob(nil), r.Documentation...)
	sort.Slice(docs, func(i, j int) bool {
		if docs[i].Path == docs[j].Path {
			return docs[i].Digest < docs[j].Digest
		}
		return docs[i].Path < docs[j].Path
	})
	for i := range docs {
		if docs[i] != r.Documentation[i] || (i > 0 && docs[i-1].Path == docs[i].Path) {
			return ErrReceiptMismatch
		}
	}
	digest := sha256.Sum256(r.canonical())
	if r.Digest != "sha256:"+hex.EncodeToString(digest[:]) {
		return ErrReceiptMismatch
	}
	return nil
}

func canonicalPaths(paths []string) ([]string, error) {
	result := append([]string(nil), paths...)
	sort.Strings(result)
	for i := 1; i < len(result); i++ {
		if result[i-1] == result[i] {
			return nil, ErrReceiptMismatch
		}
	}
	return result, nil
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (r Receipt) canonical() []byte {
	var out []byte
	add := func(value string) {
		out = append(out, strconv.AppendInt(nil, int64(len(value)), 10)...)
		out = append(out, ':')
		out = append(out, value...)
	}
	add("receipt/v1")
	add(string(r.Scope.Kind))
	add(r.Scope.Range)
	add(r.Evidence)
	add(string(r.Outcome))
	add(r.Versions.Schema)
	add(r.Versions.Tool)
	add(fmt.Sprint(len(r.ChangedPaths)))
	for _, path := range r.ChangedPaths {
		add(path)
	}
	add(fmt.Sprint(len(r.Documentation)))
	for _, doc := range r.Documentation {
		add(doc.Path)
		add(doc.Digest)
	}
	return out
}
