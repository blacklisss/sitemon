package config

import (
	"encoding/json"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestConfigUnmarshalYAMLAcceptsLegacyStringDomains(t *testing.T) {
	var cfg Config

	data := []byte(`
domains:
  - https://example.com
`)

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.Domains) != 1 {
		t.Fatalf("expected one domain, got %d", len(cfg.Domains))
	}
	if cfg.Domains[0].URL != "https://example.com" {
		t.Fatalf("unexpected url: %s", cfg.Domains[0].URL)
	}
	if cfg.Domains[0].CertificateExpiryWarningDays != 0 {
		t.Fatalf("expected zero custom warning days, got %d", cfg.Domains[0].CertificateExpiryWarningDays)
	}
}

func TestConfigUnmarshalYAMLAceptsObjectDomains(t *testing.T) {
	var cfg Config

	data := []byte(`
domains:
  - url: https://example.com
    certificateExpiryWarningDays: 5
`)

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.Domains) != 1 {
		t.Fatalf("expected one domain, got %d", len(cfg.Domains))
	}
	if cfg.Domains[0].URL != "https://example.com" {
		t.Fatalf("unexpected url: %s", cfg.Domains[0].URL)
	}
	if cfg.Domains[0].CertificateExpiryWarningDays != 5 {
		t.Fatalf("expected custom warning days, got %d", cfg.Domains[0].CertificateExpiryWarningDays)
	}
}

func TestConfigUnmarshalJSONAcceptsObjectDomains(t *testing.T) {
	var cfg Config

	data := []byte(`{"domains":[{"url":"https://example.com","certificateExpiryWarningDays":3}]}`)

	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.Domains) != 1 {
		t.Fatalf("expected one domain, got %d", len(cfg.Domains))
	}
	if cfg.Domains[0].CertificateExpiryWarningDays != 3 {
		t.Fatalf("expected custom warning days, got %d", cfg.Domains[0].CertificateExpiryWarningDays)
	}
}

func TestValidateURLRejectsNegativeCertificateWarningDays(t *testing.T) {
	if validateURL([]Domain{{URL: "https://example.com", CertificateExpiryWarningDays: -1}}) {
		t.Fatal("expected validation failure")
	}
}
