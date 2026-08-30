package site

import (
	"fmt"
	"math"
	"net/http"
	"time"
)

const (
	DefaultCode         = -1
	FailureCode         = http.StatusServiceUnavailable
	FailureThreshold    = 3
	HealthyResponseCode = http.StatusOK
)

type EventType string

const (
	EventNone              EventType = "none"
	EventDown              EventType = "down"
	EventRecovered         EventType = "recovered"
	EventCertificateExpiry EventType = "certificate_expiry"
)

type Event struct {
	Type EventType
	Text string
}

type CheckResult struct {
	ResponseCode        int
	CertificateNotAfter *time.Time
}

type Status struct {
	URL             string
	ResponseCode    int
	OldResponseCode int
	ErrorCount      uint
	IsDown          bool

	lastCertificateNotAfter time.Time
	certificateAlertSent    bool
}

func NewStatus(url string) *Status {
	return &Status{
		URL:             url,
		ResponseCode:    DefaultCode,
		OldResponseCode: HealthyResponseCode,
		ErrorCount:      0,
	}
}

func (s *Status) ApplyResponse(code int, at time.Time) Event {
	s.ResponseCode = code

	if s.ResponseCode != HealthyResponseCode && (s.OldResponseCode != s.ResponseCode || s.ErrorCount > 0) {
		s.OldResponseCode = s.ResponseCode
		s.ErrorCount++

		if s.ErrorCount == FailureThreshold {
			s.IsDown = true
			return Event{
				Type: EventDown,
				Text: fmt.Sprintf("Server down. Status %d in url: %s at %s", s.ResponseCode, s.URL, at.Format("2006-01-02 15:04:05")),
			}
		}

		return Event{Type: EventNone}
	}

	if s.OldResponseCode != s.ResponseCode {
		s.OldResponseCode = s.ResponseCode

		if s.IsDown && s.ResponseCode == HealthyResponseCode {
			event := Event{
				Type: EventRecovered,
				Text: fmt.Sprintf("Server started up in url: %s at %s", s.URL, at.Format("2006-01-02 15:04:05")),
			}
			s.ErrorCount = 0
			s.IsDown = false
			return event
		}

		s.ErrorCount = 0
	}

	return Event{Type: EventNone}
}

func (s *Status) ApplyCertificateExpiry(notAfter *time.Time, at time.Time, threshold time.Duration) Event {
	if notAfter == nil {
		return Event{Type: EventNone}
	}

	if s.lastCertificateNotAfter.IsZero() || !s.lastCertificateNotAfter.Equal(*notAfter) {
		s.lastCertificateNotAfter = *notAfter
		s.certificateAlertSent = false
	}

	if notAfter.Sub(at) > threshold {
		s.certificateAlertSent = false
		return Event{Type: EventNone}
	}

	if s.certificateAlertSent {
		return Event{Type: EventNone}
	}

	s.certificateAlertSent = true

	if !notAfter.After(at) {
		return Event{
			Type: EventCertificateExpiry,
			Text: fmt.Sprintf("TLS certificate expired for url: %s at %s (expired at %s)", s.URL, at.Format("2006-01-02 15:04:05"), notAfter.Format("2006-01-02 15:04:05")),
		}
	}

	daysLeft := int(math.Ceil(notAfter.Sub(at).Hours() / 24))

	return Event{
		Type: EventCertificateExpiry,
		Text: fmt.Sprintf("TLS certificate for url: %s expires in %d day(s) at %s", s.URL, daysLeft, notAfter.Format("2006-01-02 15:04:05")),
	}
}
