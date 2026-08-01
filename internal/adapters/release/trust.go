// Package release provides trust and archive primitives without installation lifecycle effects.
package release

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"
)

var (
	ErrManifestVerification = errors.New("manifest_verification")
	ErrRecoveryRequired     = errors.New("recovery_required")
	ErrUnsupportedPlatform  = errors.New("unsupported_platform")
	ErrDownloadBounds       = errors.New("download_bounds")
	ErrUnsafeArchive        = errors.New("unsafe_archive")
)

type Artifact struct {
	Version string `json:"version"`
	OS      string `json:"os"`
	Arch    string `json:"arch"`
	URL     string `json:"url"`
	Size    int64  `json:"size"`
	SHA256  string `json:"sha256"`
}

type Manifest struct {
	Schema    int        `json:"schema"`
	KeyID     string     `json:"key_id"`
	Issued    time.Time  `json:"issued"`
	Expires   time.Time  `json:"expires"`
	Artifacts []Artifact `json:"artifacts"`
}

type SignedManifest struct {
	Manifest  Manifest `json:"manifest"`
	Signature string   `json:"signature"`
}

type TrustPolicy struct {
	Keys         map[string]ed25519.PublicKey
	PrimaryKeyID string
	Revoked      map[string]bool
	Overlap      map[string]time.Time
	Clock        func() time.Time
}

func VerifyManifest(raw []byte, policy TrustPolicy) (Manifest, error) {
	var signed SignedManifest
	if json.Unmarshal(raw, &signed) != nil || signed.Manifest.Schema != 1 || signed.Manifest.KeyID == "" || signed.Manifest.Expires.IsZero() || signed.Manifest.Issued.IsZero() || !signed.Manifest.Expires.After(signed.Manifest.Issued) {
		return Manifest{}, ErrManifestVerification
	}
	if policy.Revoked[signed.Manifest.KeyID] {
		return Manifest{}, ErrRecoveryRequired
	}
	key := policy.Keys[signed.Manifest.KeyID]
	now := time.Now
	if policy.Clock != nil {
		now = policy.Clock
	}
	primary := policy.PrimaryKeyID
	if primary == "" {
		primary = "current"
	}
	if len(key) != ed25519.PublicKeySize || (signed.Manifest.KeyID != primary && (policy.Overlap == nil || !policy.Overlap[signed.Manifest.KeyID].After(now()))) {
		return Manifest{}, ErrManifestVerification
	}
	b, err := json.Marshal(signed.Manifest)
	if err != nil {
		return Manifest{}, ErrManifestVerification
	}
	sig, err := hex.DecodeString(signed.Signature)
	if err != nil || !ed25519.Verify(key, b, sig) || !signed.Manifest.Expires.After(now()) {
		return Manifest{}, ErrManifestVerification
	}
	for _, a := range signed.Manifest.Artifacts {
		if !validArtifact(a) {
			return Manifest{}, ErrManifestVerification
		}
	}
	return signed.Manifest, nil
}

func validArtifact(a Artifact) bool {
	if a.Version == "" || a.Size < 0 || len(a.SHA256) != 64 || (a.OS != "linux" && a.OS != "darwin") || (a.Arch != "amd64" && a.Arch != "arm64") {
		return false
	}
	_, err := hex.DecodeString(a.SHA256)
	return err == nil
}

func SelectArtifact(m Manifest, osName, arch string) (Artifact, error) {
	if (osName != "linux" && osName != "darwin") || (arch != "amd64" && arch != "arm64") {
		return Artifact{}, ErrUnsupportedPlatform
	}
	for _, a := range m.Artifacts {
		if a.OS == osName && a.Arch == arch {
			return a, nil
		}
	}
	return Artifact{}, ErrUnsupportedPlatform
}

func RecoveryDiagnosis(err error) string {
	if errors.Is(err, ErrRecoveryRequired) {
		return "trusted signing key is revoked; use a manually verified binary"
	}
	return ""
}
