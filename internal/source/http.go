package source

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"
)

// metadataHostnames are cloud-instance metadata hostnames, always blocked
// on Fetch regardless of BlockPrivateHosts (prompt-injection exfiltration).
var metadataHostnames = map[string]bool{
	"metadata.google.internal":   true,
	"metadata.aws.internal":      true,
	"metadata.azure.internal":    true,
	"instance-data":              true,
	"instance-data.ec2.internal": true,
	"metadata.alibabacloud.com":  true,
}

// Fetch returns the body of a URL over HTTP(S), bounded by a timeout and a
// size cap, and guarded against SSRF (metadata endpoints always blocked;
// private hosts blocked when BlockPrivateHosts is set).
func (s Source) Fetch(ctx context.Context, rawURL string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("source: parse url: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("source: scheme %q not allowed", u.Scheme)
	}
	host := u.Hostname()
	if host == "" {
		return "", errors.New("source: url has no host")
	}
	if isBlockedMetadataHost(host) {
		return "", fmt.Errorf("source: metadata host %q blocked", host)
	}
	if s.BlockPrivateHosts {
		if blocked, err := isPrivateHost(host); err != nil {
			return "", fmt.Errorf("source: resolve host %q: %w", host, err)
		} else if blocked {
			return "", fmt.Errorf("source: private host %q blocked", host)
		}
	}

	client := s.Client
	if client == nil {
		client = &http.Client{Timeout: s.timeout()}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", fmt.Errorf("source: build request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("source: fetch %s: %w", rawURL, err)
	}
	defer resp.Body.Close()

	max := s.maxBytes()
	data, err := io.ReadAll(io.LimitReader(resp.Body, max+1))
	if err != nil {
		return "", fmt.Errorf("source: read body: %w", err)
	}
	if int64(len(data)) > max {
		return "", ErrTooLarge
	}
	return string(data), nil
}

// timeout returns the effective Fetch timeout.
func (s Source) timeout() time.Duration {
	if s.Timeout > 0 {
		return s.Timeout
	}
	return DefaultTimeout
}

// isBlockedMetadataHost reports whether host is a cloud-metadata endpoint
// (by hostname or by link-local/Alibaba metadata IP).
func isBlockedMetadataHost(host string) bool {
	if metadataHostnames[host] {
		return true
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	if ip.IsLinkLocalUnicast() {
		return true // 169.254.0.0/16 (AWS/GCP/Azure metadata)
	}
	if ip4 := ip.To4(); ip4 != nil && ip4.Equal(net.IPv4(100, 100, 100, 200)) {
		return true // Alibaba Cloud metadata
	}
	return false
}

// isPrivateHost reports whether host resolves to a private/loopback/
// link-local address (used when BlockPrivateHosts is set).
func isPrivateHost(host string) (bool, error) {
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast(), nil
	}
	addrs, err := net.LookupHost(host)
	if err != nil {
		return false, err
	}
	for _, a := range addrs {
		if ip := net.ParseIP(a); ip != nil && (ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast()) {
			return true, nil
		}
	}
	return false, nil
}
