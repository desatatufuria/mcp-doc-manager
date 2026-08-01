package release

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/url"
)

func Download(ctx context.Context, client *http.Client, artifact Artifact, allowedHosts map[string]bool, limit int64) ([]byte, error) {
	u, err := url.Parse(artifact.URL)
	if err != nil || u.Scheme != "https" || u.Host == "" || !allowedHosts[u.Host] || artifact.Size < 0 {
		return nil, ErrManifestVerification
	}
	if artifact.Size > limit {
		return nil, ErrDownloadBounds
	}
	if client == nil {
		client = http.DefaultClient
	}
	copyClient := *client
	copyClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, artifact.URL, nil)
	if err != nil {
		return nil, ErrManifestVerification
	}
	resp, err := copyClient.Do(req)
	if err != nil {
		return nil, ErrManifestVerification
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, ErrManifestVerification
	}
	if resp.ContentLength > limit || resp.ContentLength > artifact.Size {
		return nil, ErrDownloadBounds
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil || int64(len(b)) > limit || int64(len(b)) != artifact.Size {
		return nil, ErrDownloadBounds
	}
	sum := sha256.Sum256(b)
	if hex.EncodeToString(sum[:]) != artifact.SHA256 {
		return nil, ErrManifestVerification
	}
	return b, nil
}
