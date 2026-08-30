package httpcheck

import (
	"context"
	"net"
	"net/http"
	"site_monitoring/internal/domain/site"
	"time"
)

const userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0 Safari/537.36"

type Checker struct {
	client *http.Client
}

func NewChecker(timeout time.Duration) *Checker {
	dialer := &net.Dialer{Timeout: timeout}

	return &Checker{
		client: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
					return dialer.DialContext(ctx, "tcp4", address)
				},
			},
		},
	}
}

func (c *Checker) Check(ctx context.Context, url string) (site.CheckResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return site.CheckResult{}, err
	}
	req.Header.Set("User-Agent", userAgent)

	res, err := c.client.Do(req)
	if err != nil {
		return site.CheckResult{}, err
	}
	defer res.Body.Close()

	result := site.CheckResult{ResponseCode: res.StatusCode}
	if res.TLS != nil && len(res.TLS.PeerCertificates) > 0 {
		notAfter := res.TLS.PeerCertificates[0].NotAfter
		result.CertificateNotAfter = &notAfter
	}

	return result, nil
}
