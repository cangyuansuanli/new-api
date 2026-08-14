package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/relay/imagevendor"
)

func sameImagePublicOrigin(rawURL, publicBase string) bool {
	imageURL, imageErr := url.Parse(strings.TrimSpace(rawURL))
	baseURL, baseErr := url.Parse(strings.TrimSpace(publicBase))
	if imageErr != nil || baseErr != nil || imageURL.Scheme != "https" || baseURL.Scheme != "https" || !strings.EqualFold(imageURL.Host, baseURL.Host) {
		return false
	}
	basePath := strings.TrimRight(baseURL.Path, "/")
	return basePath == "" || imageURL.Path == basePath || strings.HasPrefix(imageURL.Path, basePath+"/")
}

func trustedTaskImageResultURL(originModel, rawURL string) bool {
	cfg := getR2Config()
	if cfg != nil && sameImagePublicOrigin(rawURL, cfg.PublicBase) {
		return true
	}
	policy := imagevendor.ResolveRehostPolicy(originModel)
	return policy.TrustPublicURL != nil && policy.TrustPublicURL(strings.TrimSpace(rawURL))
}

// DownloadTaskImageResult stages a trusted queued result on disk so the API
// can stream b64_json without retaining the binary and encoded copies in RAM.
func DownloadTaskImageResult(ctx context.Context, originModel, rawURL string) (*os.File, error) {
	if !trustedTaskImageResultURL(originModel, rawURL) {
		return nil, fmt.Errorf("untrusted queued image result URL")
	}
	baseClient := GetHttpClient()
	client := &http.Client{Timeout: 600 * time.Second}
	if baseClient != nil {
		client.Transport = baseClient.Transport
	}
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return fmt.Errorf("too many queued image result redirects")
		}
		if !trustedTaskImageResultURL(originModel, req.URL.String()) {
			return fmt.Errorf("untrusted queued image result redirect")
		}
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download queued image result: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download queued image result HTTP %d", resp.StatusCode)
	}
	if resp.ContentLength > constantMaxGeneratedImageBytes {
		return nil, fmt.Errorf("queued image result exceeds %d bytes", constantMaxGeneratedImageBytes)
	}
	file, err := os.CreateTemp("", "new-api-image-result-*")
	if err != nil {
		return nil, err
	}
	cleanup := func() {
		name := file.Name()
		_ = file.Close()
		_ = os.Remove(name)
	}
	written, err := io.Copy(file, io.LimitReader(resp.Body, int64(constantMaxGeneratedImageBytes)+1))
	if err != nil {
		cleanup()
		return nil, err
	}
	if written > constantMaxGeneratedImageBytes {
		cleanup()
		return nil, fmt.Errorf("queued image result exceeds %d bytes", constantMaxGeneratedImageBytes)
	}
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		cleanup()
		return nil, err
	}
	return file, nil
}
