package probe

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// probeIMDS attempts AWS IMDSv2 and returns a short identity string.
// Soft-fails on any error (not on AWS, hop limit, timeout, etc.).
// Uses a 200ms timeout and dialer hop limit (TTL) of 1 when the OS allows.
func probeIMDS() string {
	dialer := &net.Dialer{
		Timeout: 200 * time.Millisecond,
		Control: imdsDialControl,
	}
	client := &http.Client{
		Timeout: 200 * time.Millisecond,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return dialer.DialContext(ctx, network, addr)
			},
			DisableKeepAlives: true,
		},
	}
	req, err := http.NewRequest(http.MethodPut, "http://169.254.169.254/latest/api/token", nil)
	if err != nil {
		return ""
	}
	req.Header.Set("X-aws-ec2-metadata-token-ttl-seconds", "60")
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}
	token, err := io.ReadAll(io.LimitReader(resp.Body, 256))
	if err != nil || len(token) == 0 {
		return ""
	}

	get := func(path string) string {
		r, err := http.NewRequest(http.MethodGet, "http://169.254.169.254"+path, nil)
		if err != nil {
			return ""
		}
		r.Header.Set("X-aws-ec2-metadata-token", string(token))
		res, err := client.Do(r)
		if err != nil {
			return ""
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			return ""
		}
		b, err := io.ReadAll(io.LimitReader(res.Body, 256))
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(b))
	}

	id := get("/latest/meta-data/instance-id")
	itype := get("/latest/meta-data/instance-type")
	if id == "" && itype == "" {
		return ""
	}
	parts := []string{}
	if id != "" {
		parts = append(parts, id)
	}
	if itype != "" {
		parts = append(parts, itype)
	}
	return strings.Join(parts, " ")
}
