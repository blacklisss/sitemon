package monitor

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"site_monitoring/internal/domain/site"

	"github.com/sirupsen/logrus"
)

type stubChecker struct {
	mu      sync.Mutex
	results []site.CheckResult
	errs    []error
	idx     int
}

func (s *stubChecker) Check(_ context.Context, _ string) (site.CheckResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.idx >= len(s.results) {
		return site.CheckResult{ResponseCode: http.StatusOK}, nil
	}

	result := s.results[s.idx]
	var err error
	if s.idx < len(s.errs) {
		err = s.errs[s.idx]
	}
	s.idx++

	return result, err
}

type stubNotifier struct {
	mu       sync.Mutex
	messages []string
}

func (s *stubNotifier) SendMessage(message string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages = append(s.messages, message)
	return nil
}

func (s *stubNotifier) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.messages)
}

func (s *stubNotifier) messagesCopy() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]string, len(s.messages))
	copy(out, s.messages)

	return out
}

func TestService_CheckOnceSendsDownOnThirdFailure(t *testing.T) {
	checker := &stubChecker{results: []site.CheckResult{
		{ResponseCode: http.StatusInternalServerError},
		{ResponseCode: http.StatusInternalServerError},
		{ResponseCode: http.StatusInternalServerError},
		{ResponseCode: http.StatusInternalServerError},
	}}
	notifier := &stubNotifier{}
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)

	svc := NewService(checker, notifier, logger, time.Millisecond)
	ctx := context.Background()
	now := time.Date(2026, 3, 6, 12, 0, 0, 0, time.UTC)
	target := Target{URL: "https://example.com"}

	svc.checkOnce(ctx, target, now)
	svc.checkOnce(ctx, target, now.Add(time.Minute))
	svc.checkOnce(ctx, target, now.Add(2*time.Minute))

	if got := notifier.count(); got != 1 {
		t.Fatalf("expected one notification, got %d", got)
	}
}

func TestService_CheckOnceSuppressesDownWhenConfirmationIsHealthy(t *testing.T) {
	checker := &stubChecker{results: []site.CheckResult{
		{ResponseCode: http.StatusInternalServerError},
		{ResponseCode: http.StatusInternalServerError},
		{ResponseCode: http.StatusInternalServerError},
		{ResponseCode: http.StatusOK},
	}}
	notifier := &stubNotifier{}
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)

	svc := NewService(checker, notifier, logger, time.Millisecond)
	ctx := context.Background()
	now := time.Date(2026, 3, 6, 12, 0, 0, 0, time.UTC)
	target := Target{URL: "https://example.com"}

	svc.checkOnce(ctx, target, now)
	svc.checkOnce(ctx, target, now.Add(time.Minute))
	svc.checkOnce(ctx, target, now.Add(2*time.Minute))

	if got := notifier.count(); got != 0 {
		t.Fatalf("CheckOnce(recovered on confirmation) sent %d notifications, want 0", got)
	}
}

func TestService_CheckOnceSendsRecoveryAfterDown(t *testing.T) {
	checker := &stubChecker{results: []site.CheckResult{
		{ResponseCode: http.StatusInternalServerError},
		{ResponseCode: http.StatusInternalServerError},
		{ResponseCode: http.StatusInternalServerError},
		{ResponseCode: http.StatusInternalServerError},
		{ResponseCode: http.StatusOK},
	}}
	notifier := &stubNotifier{}
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)

	svc := NewService(checker, notifier, logger, time.Millisecond)
	ctx := context.Background()
	now := time.Date(2026, 3, 6, 12, 0, 0, 0, time.UTC)
	target := Target{URL: "https://example.com"}

	svc.checkOnce(ctx, target, now)
	svc.checkOnce(ctx, target, now.Add(time.Minute))
	svc.checkOnce(ctx, target, now.Add(2*time.Minute))
	svc.checkOnce(ctx, target, now.Add(3*time.Minute))

	if got := notifier.count(); got != 2 {
		t.Fatalf("expected two notifications (down + recovery), got %d", got)
	}
}

func TestService_CheckOnceSendsRecoveryAfterRestart(t *testing.T) {
	stateFile := t.TempDir() + "/state.json"
	now := time.Date(2026, 3, 6, 12, 0, 0, 0, time.UTC)
	target := Target{URL: "https://example.com"}

	firstChecker := &stubChecker{results: []site.CheckResult{
		{ResponseCode: http.StatusInternalServerError},
		{ResponseCode: http.StatusInternalServerError},
		{ResponseCode: http.StatusInternalServerError},
		{ResponseCode: http.StatusInternalServerError},
	}}
	firstNotifier := &stubNotifier{}
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)

	firstSvc := NewService(firstChecker, firstNotifier, logger, time.Millisecond)
	if err := firstSvc.UseStateFile(stateFile); err != nil {
		t.Fatalf("UseStateFile(%q) = %v, want nil", stateFile, err)
	}

	firstSvc.checkOnce(context.Background(), target, now)
	firstSvc.checkOnce(context.Background(), target, now.Add(time.Minute))
	firstSvc.checkOnce(context.Background(), target, now.Add(2*time.Minute))

	secondChecker := &stubChecker{results: []site.CheckResult{{ResponseCode: http.StatusOK}}}
	secondNotifier := &stubNotifier{}
	secondSvc := NewService(secondChecker, secondNotifier, logger, time.Millisecond)
	if err := secondSvc.UseStateFile(stateFile); err != nil {
		t.Fatalf("UseStateFile(%q) after restart = %v, want nil", stateFile, err)
	}

	secondSvc.checkOnce(context.Background(), target, now.Add(3*time.Minute))

	messages := secondNotifier.messagesCopy()
	if got, want := len(messages), 1; got != want {
		t.Fatalf("CheckOnce(recovery after restart) sent %d notifications, want %d", got, want)
	}
	if !strings.Contains(messages[0], "Server started up") {
		t.Fatalf("CheckOnce(recovery after restart) message = %q, want Server started up", messages[0])
	}
}

func TestService_CheckOnceMapsCheckerErrorToFailureCode(t *testing.T) {
	checker := &stubChecker{
		results: []site.CheckResult{
			{ResponseCode: 0},
			{ResponseCode: 0},
			{ResponseCode: 0},
			{ResponseCode: 0},
		},
		errs: []error{errors.New("network fail"), errors.New("network fail"), errors.New("network fail"), errors.New("network fail")},
	}
	notifier := &stubNotifier{}
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)

	svc := NewService(checker, notifier, logger, time.Millisecond)
	ctx := context.Background()
	now := time.Date(2026, 3, 6, 12, 0, 0, 0, time.UTC)
	target := Target{URL: "https://example.com"}

	svc.checkOnce(ctx, target, now)
	svc.checkOnce(ctx, target, now.Add(time.Minute))
	svc.checkOnce(ctx, target, now.Add(2*time.Minute))

	if got := notifier.count(); got != 1 {
		t.Fatalf("expected one down notification after repeated checker errors, got %d", got)
	}
}

func TestService_CheckOnceSendsCertificateExpiryAlertOnce(t *testing.T) {
	now := time.Date(2026, 3, 6, 12, 0, 0, 0, time.UTC)
	notAfter := now.Add(10 * 24 * time.Hour)

	checker := &stubChecker{results: []site.CheckResult{
		{ResponseCode: http.StatusOK, CertificateNotAfter: &notAfter},
		{ResponseCode: http.StatusOK, CertificateNotAfter: &notAfter},
	}}
	notifier := &stubNotifier{}
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)

	svc := NewService(checker, notifier, logger, time.Millisecond)
	ctx := context.Background()
	target := Target{URL: "https://example.com"}

	svc.checkOnce(ctx, target, now)
	svc.checkOnce(ctx, target, now.Add(time.Hour))

	if got := notifier.count(); got != 1 {
		t.Fatalf("expected one certificate expiry notification, got %d", got)
	}

	if messages := notifier.messagesCopy(); !strings.Contains(messages[0], "TLS certificate") {
		t.Fatalf("expected certificate expiry message, got %q", messages[0])
	}
}

func TestService_CheckOnceUsesPerTargetCertificateThreshold(t *testing.T) {
	now := time.Date(2026, 3, 6, 12, 0, 0, 0, time.UTC)
	notAfter := now.Add(7 * 24 * time.Hour)

	checker := &stubChecker{results: []site.CheckResult{
		{ResponseCode: http.StatusOK, CertificateNotAfter: &notAfter},
		{ResponseCode: http.StatusOK, CertificateNotAfter: &notAfter},
	}}
	notifier := &stubNotifier{}
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)

	svc := NewService(checker, notifier, logger, time.Millisecond)
	ctx := context.Background()

	svc.checkOnce(ctx, Target{
		URL:                             "https://example.com",
		CertificateExpiryAlertThreshold: 5 * 24 * time.Hour,
	}, now)

	if got := notifier.count(); got != 0 {
		t.Fatalf("expected no notification before custom threshold, got %d", got)
	}

	svc.checkOnce(ctx, Target{
		URL:                             "https://example.com",
		CertificateExpiryAlertThreshold: 8 * 24 * time.Hour,
	}, now.Add(time.Hour))

	if got := notifier.count(); got != 1 {
		t.Fatalf("expected notification after custom threshold, got %d", got)
	}
}
