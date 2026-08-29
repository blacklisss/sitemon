package httpcheck

import (
	"context"
	"net/http"
	"site_monitoring/internal/domain/site"
	"time"
)

type Checker struct {
	client *http.Client
}

func NewChecker(timeout time.Duration) *Checker {
	return &Checker{
		client: &http.Client{Timeout: timeout},
	}
}

func (c *Checker) Check(ctx context.Context, url string) (site.CheckResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return site.CheckResult{}, err
	}

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
