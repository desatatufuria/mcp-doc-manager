package release

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func signed(t *testing.T, key ed25519.PrivateKey, m Manifest) []byte {
	t.Helper()
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	s := ed25519.Sign(key, b)
	b, err = json.Marshal(SignedManifest{Manifest: m, Signature: hex.EncodeToString(s)})
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestTrust(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	m := Manifest{Schema: 1, KeyID: "current", Issued: now.Add(-time.Hour), Expires: now.Add(time.Hour), Artifacts: []Artifact{{Version: "v1", OS: "linux", Arch: "arm64", URL: "https://releases.example/a", Size: 3, SHA256: "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"}}}
	policy := TrustPolicy{Keys: map[string]ed25519.PublicKey{"current": pub}, Clock: func() time.Time { return now }}

	t.Run("accepts canonical signature and platform", func(t *testing.T) {
		got, err := VerifyManifest(signed(t, priv, m), policy)
		if err != nil {
			t.Fatal(err)
		}
		a, err := SelectArtifact(got, "linux", "arm64")
		if err != nil || a.Version != "v1" {
			t.Fatalf("artifact=%+v err=%v", a, err)
		}
	})
	for _, tc := range []struct {
		name string
		raw  []byte
		p    TrustPolicy
		want error
	}{
		{"bad-signature", func() []byte { b := signed(t, priv, m); b[len(b)-2] ^= 1; return b }(), policy, ErrManifestVerification},
		{"expired", signed(t, priv, func() Manifest { x := m; x.Expires = now.Add(-time.Second); return x }()), policy, ErrManifestVerification},
		{"revoked-recovery", signed(t, priv, m), TrustPolicy{Keys: policy.Keys, Revoked: map[string]bool{"current": true}, Clock: policy.Clock}, ErrRecoveryRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := VerifyManifest(tc.raw, tc.p)
			if !errors.Is(err, tc.want) {
				t.Fatalf("err=%v want %v", err, tc.want)
			}
			if tc.want == ErrRecoveryRequired && RecoveryDiagnosis(err) == "" {
				t.Fatal("missing recovery diagnosis")
			}
		})
	}
	t.Run("accepts explicitly overlapping replacement", func(t *testing.T) {
		p2, k2, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		x := m
		x.KeyID = "replacement"
		_, err = VerifyManifest(signed(t, k2, x), TrustPolicy{Keys: map[string]ed25519.PublicKey{"current": pub, "replacement": p2}, Overlap: map[string]time.Time{"replacement": now.Add(time.Minute)}, Clock: policy.Clock})
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("rejects unauthorized rotation and platform", func(t *testing.T) {
		p2, k2, _ := ed25519.GenerateKey(rand.Reader)
		x := m
		x.KeyID = "replacement"
		if _, err := VerifyManifest(signed(t, k2, x), TrustPolicy{Keys: map[string]ed25519.PublicKey{"replacement": p2}, Clock: policy.Clock}); !errors.Is(err, ErrManifestVerification) {
			t.Fatalf("err=%v", err)
		}
		if _, err := SelectArtifact(m, "windows", "amd64"); !errors.Is(err, ErrUnsupportedPlatform) {
			t.Fatalf("err=%v", err)
		}
	})
}

func TestDownloadRejectsUntrustedResponses(t *testing.T) {
	good := []byte("abc")
	sum := sha256.Sum256(good)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/redirect":
			http.Redirect(w, r, "/artifact", http.StatusFound)
		case "/large":
			w.Write([]byte("abcd"))
		default:
			w.Write(good)
		}
	}))
	defer server.Close()
	client := server.Client()
	host := server.URL[len("https://"):]
	base := Artifact{URL: server.URL + "/artifact", Size: 3, SHA256: hex.EncodeToString(sum[:])}
	for _, tc := range []struct {
		name  string
		a     Artifact
		hosts map[string]bool
		want  error
	}{
		{"host", base, map[string]bool{}, ErrManifestVerification},
		{"redirect", func() Artifact { x := base; x.URL = server.URL + "/redirect"; return x }(), map[string]bool{host: true}, ErrManifestVerification},
		{"size", func() Artifact { x := base; x.URL = server.URL + "/large"; x.Size = 4; return x }(), map[string]bool{host: true}, ErrDownloadBounds},
		{"digest", func() Artifact { x := base; x.SHA256 = "00" + base.SHA256[2:]; return x }(), map[string]bool{host: true}, ErrManifestVerification},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Download(context.Background(), client, tc.a, tc.hosts, 3)
			if !errors.Is(err, tc.want) {
				t.Fatalf("err=%v want %v", err, tc.want)
			}
		})
	}
	if got, err := Download(context.Background(), client, base, map[string]bool{host: true}, 3); err != nil || string(got) != "abc" {
		t.Fatalf("got=%q err=%v", got, err)
	}
}
