package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"testing"
)

func TestReceiptDigestIsCanonicalAndContentBound(t *testing.T) {
	base := Report{
		Outcome:  OutcomeUpdate,
		Evidence: Evidence{Scope: Scope{Kind: ScopeRange, Range: "a..b"}, Identity: "sha256:evidence", ChangedPaths: []string{"z.md", "a.md"}},
	}
	versions := ReceiptVersions{Schema: "receipt/v1", Tool: "1.0.0"}
	first, err := NewReceipt(base, []DocumentationBlob{{Path: "docs/z.md", Digest: "sha256:z"}, {Path: "README.md", Digest: "sha256:a"}}, versions)
	if err != nil {
		t.Fatal(err)
	}
	if want := independentDigest(first); first.Digest != want {
		t.Fatalf("digest = %q, oracle = %q", first.Digest, want)
	}
	if first.Digest != "sha256:6cd7bb3b6b26c6f29ede6f08817142b7607989f8df858124443459ec6159ac55" {
		t.Fatalf("digest golden = %q", first.Digest)
	}
	semanticallyEqual := base
	semanticallyEqual.Evidence.ChangedPaths = []string{"a.md", "z.md"}
	second, err := NewReceipt(semanticallyEqual, []DocumentationBlob{{Path: "README.md", Digest: "sha256:a"}, {Path: "docs/z.md", Digest: "sha256:z"}}, versions)
	if err != nil || first.Digest != second.Digest {
		t.Fatalf("equal receipts = %#v, %#v, %v", first, second, err)
	}
	for _, changed := range []Report{
		{Outcome: OutcomeUpdate, Evidence: Evidence{Scope: Scope{Kind: ScopeRange, Range: "b..c"}, Identity: base.Evidence.Identity, ChangedPaths: base.Evidence.ChangedPaths}},
		{Outcome: OutcomeUpdate, Evidence: Evidence{Scope: base.Evidence.Scope, Identity: "sha256:changed", ChangedPaths: base.Evidence.ChangedPaths}},
	} {
		receipt, err := NewReceipt(changed, first.Documentation, versions)
		if err != nil || receipt.Digest == first.Digest {
			t.Fatalf("changed receipt = %#v, %v", receipt, err)
		}
	}
	changedVersion, err := NewReceipt(base, first.Documentation, ReceiptVersions{Schema: "receipt/v2", Tool: versions.Tool})
	if err != nil || changedVersion.Digest == first.Digest {
		t.Fatalf("changed version receipt = %#v, %v", changedVersion, err)
	}
	for _, mutate := range []func(*Receipt){
		func(r *Receipt) { r.Digest = "sha256:other" },
		func(r *Receipt) { r.Scope.Range = "other..range" },
		func(r *Receipt) { r.Scope.Kind = ScopeStaged },
		func(r *Receipt) { r.Evidence = "sha256:other" },
		func(r *Receipt) { r.ChangedPaths[0] = "other.md" },
		func(r *Receipt) { r.Documentation[0].Path = "docs/other.md" },
		func(r *Receipt) { r.Documentation[0].Digest = "sha256:other" },
		func(r *Receipt) { r.Outcome = OutcomeCreate },
		func(r *Receipt) { r.Versions.Tool = "docmanager/2" },
		func(r *Receipt) { r.Versions.Schema = "receipt/v2" },
	} {
		tampered := first
		tampered.ChangedPaths = append([]string(nil), first.ChangedPaths...)
		tampered.Documentation = append([]DocumentationBlob(nil), first.Documentation...)
		mutate(&tampered)
		if err := tampered.ValidateDigest(); !errors.Is(err, ErrReceiptMismatch) {
			t.Fatalf("tampered receipt error = %v", err)
		}
	}
	if _, err := NewReceipt(base, []DocumentationBlob{{Path: "README.md", Digest: "a"}, {Path: "README.md", Digest: "b"}}, versions); !errors.Is(err, ErrReceiptMismatch) {
		t.Fatalf("duplicate documentation error = %v", err)
	}
	duplicate := first
	duplicate.ChangedPaths = append([]string{"a.md"}, first.ChangedPaths...)
	if err := duplicate.ValidateDigest(); !errors.Is(err, ErrReceiptMismatch) {
		t.Fatalf("duplicate changed paths error = %v", err)
	}
}

func independentDigest(r Receipt) string {
	var canonical []byte
	add := func(value string) {
		canonical = append(canonical, strconv.AppendInt(nil, int64(len(value)), 10)...)
		canonical = append(canonical, ':')
		canonical = append(canonical, value...)
	}
	add("receipt/v1")
	add(string(r.Scope.Kind))
	add(r.Scope.Range)
	add(r.Evidence)
	add(string(r.Outcome))
	add(r.Versions.Schema)
	add(r.Versions.Tool)
	add(strconv.Itoa(len(r.ChangedPaths)))
	for _, path := range r.ChangedPaths {
		add(path)
	}
	add(strconv.Itoa(len(r.Documentation)))
	for _, doc := range r.Documentation {
		add(doc.Path)
		add(doc.Digest)
	}
	digest := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(digest[:])
}
