package site

import (
	"net/http"
	"testing"
	"time"
)

func TestStatus_ThirdFailureEmitsDownEvent(t *testing.T) {
	st := NewStatus("https://example.com")
	now := time.Date(2026, 3, 6, 12, 0, 0, 0, time.UTC)

	for i := 0; i < FailureThreshold-1; i++ {
		event := st.ApplyResponse(http.StatusInternalServerError, now)
		if event.Type != EventNone {
			t.Fatalf("expected no event before threshold, got %s", event.Type)
		}
	}

	event := st.ApplyResponse(http.StatusInternalServerError, now)
	if event.Type != EventDown {
		t.Fatalf("expected down event, got %s", event.Type)
	}
	if event.Text == "" {
		t.Fatal("expected down event text")
	}
	if !st.IsDown {
		t.Fatal("expected status to be marked down")
	}
}

func TestStatus_RecoveryAfterFailuresEmitsRecoveredEvent(t *testing.T) {
	st := NewStatus("https://example.com")
	now := time.Date(2026, 3, 6, 12, 0, 0, 0, time.UTC)

	for i := 0; i < FailureThreshold; i++ {
		_ = st.ApplyResponse(http.StatusInternalServerError, now)
	}

	event := st.ApplyResponse(http.StatusOK, now.Add(time.Minute))
	if event.Type != EventRecovered {
		t.Fatalf("expected recovered event, got %s", event.Type)
	}
	if event.Text == "" {
		t.Fatal("expected recovery event text")
	}
	if st.ErrorCount != 0 {
		t.Fatalf("expected error count reset, got %d", st.ErrorCount)
	}
	if st.IsDown {
		t.Fatal("expected status to be marked recovered")
	}
}

func TestStatus_FirstHealthyCheckProducesNoEvent(t *testing.T) {
	st := NewStatus("https://example.com")
	now := time.Date(2026, 3, 6, 12, 0, 0, 0, time.UTC)

	event := st.ApplyResponse(http.StatusOK, now)
	if event.Type != EventNone {
		t.Fatalf("expected no event, got %s", event.Type)
	}
	if st.ErrorCount != 0 {
		t.Fatalf("expected zero errors, got %d", st.ErrorCount)
	}
}

func TestStatus_NonHealthyStatusChangeAfterDownDoesNotRecover(t *testing.T) {
	st := NewStatus("https://example.com")
	now := time.Date(2026, 3, 6, 12, 0, 0, 0, time.UTC)

	for i := 0; i < FailureThreshold; i++ {
		_ = st.ApplyResponse(http.StatusInternalServerError, now)
	}

	event := st.ApplyResponse(http.StatusBadGateway, now.Add(time.Minute))
	if event.Type != EventNone {
		t.Fatalf("ApplyResponse(502 after down) = %s, want %s", event.Type, EventNone)
	}
	if !st.IsDown {
		t.Fatal("expected status to remain down")
	}
}

func TestStatus_CertificateExpiringSoonEmitsSingleEvent(t *testing.T) {
	st := NewStatus("https://example.com")
	now := time.Date(2026, 3, 6, 12, 0, 0, 0, time.UTC)
	notAfter := now.Add(10 * 24 * time.Hour)

	event := st.ApplyCertificateExpiry(&notAfter, now, 10*24*time.Hour)
	if event.Type != EventCertificateExpiry {
		t.Fatalf("expected certificate expiry event, got %s", event.Type)
	}
	if event.Text == "" {
		t.Fatal("expected certificate expiry event text")
	}

	event = st.ApplyCertificateExpiry(&notAfter, now.Add(time.Hour), 10*24*time.Hour)
	if event.Type != EventNone {
		t.Fatalf("expected no duplicate certificate event, got %s", event.Type)
	}
}

func TestStatus_CertificateRenewalResetsAlertState(t *testing.T) {
	st := NewStatus("https://example.com")
	now := time.Date(2026, 3, 6, 12, 0, 0, 0, time.UTC)
	firstNotAfter := now.Add(5 * 24 * time.Hour)

	_ = st.ApplyCertificateExpiry(&firstNotAfter, now, 10*24*time.Hour)

	renewedNotAfter := now.Add(45 * 24 * time.Hour)
	event := st.ApplyCertificateExpiry(&renewedNotAfter, now.Add(time.Hour), 10*24*time.Hour)
	if event.Type != EventNone {
		t.Fatalf("expected no event for renewed certificate outside threshold, got %s", event.Type)
	}

	event = st.ApplyCertificateExpiry(&renewedNotAfter, renewedNotAfter.Add(-10*24*time.Hour), 10*24*time.Hour)
	if event.Type != EventCertificateExpiry {
		t.Fatalf("expected certificate expiry event after renewal enters threshold, got %s", event.Type)
	}
}
