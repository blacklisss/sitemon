package monitor

import (
	"context"
	"errors"
	"net/url"
	"sync"
	"time"

	"site_monitoring/internal/domain/site"

	"github.com/sirupsen/logrus"
)

const DefaultCertificateExpiryAlertThreshold = 10 * 24 * time.Hour

type Target struct {
	URL                             string
	CertificateExpiryAlertThreshold time.Duration
}

type Checker interface {
	Check(ctx context.Context, url string) (site.CheckResult, error)
}

type Notifier interface {
	SendMessage(message string) error
}

type Service struct {
	checker  Checker
	notifier Notifier
	logger   *logrus.Logger
	delay    time.Duration

	mu     sync.Mutex
	states map[string]*site.Status
}

func NewService(checker Checker, notifier Notifier, logger *logrus.Logger, delay time.Duration) *Service {
	return &Service{
		checker:  checker,
		notifier: notifier,
		logger:   logger,
		delay:    delay,
		states:   make(map[string]*site.Status),
	}
}

func (s *Service) Run(ctx context.Context, targets []Target) {
	var wg sync.WaitGroup

	for _, target := range targets {
		wg.Add(1)
		go func(target Target) {
			defer wg.Done()
			s.monitorDomain(ctx, target)
		}(target)
	}

	<-ctx.Done()
	wg.Wait()
}

func (s *Service) monitorDomain(ctx context.Context, target Target) {
	s.logger.Infoln("Start checking", target.URL)

	ticker := time.NewTicker(s.delay)
	defer ticker.Stop()

	s.checkOnce(ctx, target, time.Now())

	for {
		select {
		case <-ctx.Done():
			s.logger.Warnln("stopping checks for", target.URL)
			return
		case t := <-ticker.C:
			s.checkOnce(ctx, target, t)
		}
	}
}

func (s *Service) checkOnce(ctx context.Context, target Target, at time.Time) {
	s.logger.Infof("Checking %s at %s\n", target.URL, at)

	requestCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	result, err := s.checker.Check(requestCtx, target.URL)
	if err != nil {
		if errors.Is(ctx.Err(), context.Canceled) {
			return
		}
		s.logger.Errorln("cannot do request:", err.Error())
		result.ResponseCode = site.FailureCode
	}

	st := s.getOrCreateState(target.URL)
	events := []site.Event{st.ApplyResponse(result.ResponseCode, at)}

	if isHTTPSURL(target.URL) {
		events = append(events, st.ApplyCertificateExpiry(result.CertificateNotAfter, at, target.certificateExpiryAlertThreshold()))
	}

	for _, event := range events {
		if event.Type == site.EventNone || event.Text == "" {
			continue
		}

		if err = s.notifier.SendMessage(event.Text); err != nil {
			s.logger.Errorln("cannot send notification:", err.Error())
		}
	}
}

func (s *Service) getOrCreateState(url string) *site.Status {
	s.mu.Lock()
	defer s.mu.Unlock()

	if current, ok := s.states[url]; ok {
		return current
	}

	st := site.NewStatus(url)
	s.states[url] = st

	return st
}

func (t Target) certificateExpiryAlertThreshold() time.Duration {
	if t.CertificateExpiryAlertThreshold > 0 {
		return t.CertificateExpiryAlertThreshold
	}

	return DefaultCertificateExpiryAlertThreshold
}

func isHTTPSURL(rawURL string) bool {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	return parsedURL.Scheme == "https"
}
