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
// private hosts blocked when BlockPrivateHosts is set). A non-2xx status is
// an error so a failed fetch is distinguishable from an empty source.
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

	client := s.Client
	if client == nil {
		client = &http.Client{Timeout: s.timeout(), Transport: s.transport()}
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

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("source: fetch %s: status %d", rawURL, resp.StatusCode)
	}

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

// transport returns an SSRF-aware http.Transport whose DialContext resolves
// the host ONCE, validates every address, and dials a pinned IP — so the
// client can't re-resolve to a different address (DNS-rebinding defense).
// Metadata IPs are always rejected; private/loopback are rejected when
// BlockPrivateHosts is set.
func (s Source) transport() *http.Transport {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.DialContext = s.dialContext
	return t
}

// dialContext resolves addr's host once, rejects blocked addresses, and
// dials the first validated IP directly (pinned, no re-resolution).
func (s Source) dialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	ips, err := resolveIPs(ctx, host)
	if err != nil {
		return nil, err
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("source: host %q has no addresses", host)
	}
	for _, ip := range ips {
		if isBlockedMetadataIP(ip) {
			return nil, fmt.Errorf("source: metadata host %q blocked", host)
		}
		if s.BlockPrivateHosts && (ip.IsPrivate() || ip.IsLoopback()) {
			return nil, fmt.Errorf("source: private host %q blocked", host)
		}
	}
	var d net.Dialer
	return d.DialContext(ctx, network, net.JoinHostPort(ips[0].String(), port))
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
	return isBlockedMetadataIP(ip)
}

// isBlockedMetadataIP reports whether ip is a cloud-metadata address
// (link-local 169.254.0.0/16, or Alibaba Cloud 100.100.100.200).
func isBlockedMetadataIP(ip net.IP) bool {
	if ip.IsLinkLocalUnicast() {
		return true
	}
	if ip4 := ip.To4(); ip4 != nil && ip4.Equal(net.IPv4(100, 100, 100, 200)) {
		return true
	}
	return false
}

// resolveIPs returns the IP literal, or the resolved addresses of a host.
func resolveIPs(ctx context.Context, host string) ([]net.IP, error) {
	if ip := net.ParseIP(host); ip != nil {
		return []net.IP{ip}, nil
	}
	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	ips := make([]net.IP, 0, len(addrs))
	for _, a := range addrs {
		ips = append(ips, a.IP)
	}
	return ips, nil
}
