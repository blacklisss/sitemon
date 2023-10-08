package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	t.Run("load json config", func(t *testing.T) {
		configContent := `{
			"notification": {
				"BotAPI": "test_bot_api",
				"ChatID": 123456
			},
			"domains": ["http://example.com"]
		}`
		configFile, err := os.CreateTemp("", "config_*.json")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(configFile.Name())

		if _, err := configFile.WriteString(configContent); err != nil {
			t.Fatal(err)
		}

		cfg, err := Load(configFile.Name())
		if err != nil {
			t.Fatal(err)
		}

		if cfg.Notification.BotAPI != "test_bot_api" {
			t.Errorf("expected botAPI to be test_bot_api, got %s", cfg.Notification.BotAPI)
		}
		if cfg.Notification.ChatID != 123456 {
			t.Errorf("expected chatID to be 123456, got %d", cfg.Notification.ChatID)
		}
		if len(cfg.Domains) != 1 || cfg.Domains[0] != "http://example.com" {
			t.Errorf("expected domains to be [http://example.com], got %v", cfg.Domains)
		}
	})

	t.Run("load yaml config", func(t *testing.T) {
		configContent := `
timing:
  delay: 1s
notification:
  BotAPI: test_bot_api
  ChatID: 123456
domains:
- http://example.com
`
		configFile, err := os.CreateTemp("", "config_*.yaml")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(configFile.Name())

		if _, err := configFile.WriteString(configContent); err != nil {
			t.Fatal(err)
		}

		cfg, err := Load(configFile.Name())
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Timing.Delay != time.Second {
			t.Errorf("expected delay to be 1s, got %s", cfg.Timing.Delay)
		}
		if cfg.Notification.BotAPI != "test_bot_api" {
			t.Errorf("expected botAPI to be test_bot_api, got %s", cfg.Notification.BotAPI)
		}
		if cfg.Notification.ChatID != 123456 {
			t.Errorf("expected chatID to be 123456, got %d", cfg.Notification.ChatID)
		}
		if len(cfg.Domains) != 1 || cfg.Domains[0] != "http://example.com" {
			t.Errorf("expected domains to be [http://example.com], got %v", cfg.Domains)
		}
	})
}

func TestIsURL(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"http://example.com", true},
		{"https://example.com", true},
		{"ftp://example.com", true},
		{"example.com", false},
		{"", false},
		{"/path/to/something", false},
	}

	for _, test := range tests {
		if result := isURL(test.input); result != test.expected {
			t.Errorf("for input %s, expected %v, got %v", test.input, test.expected, result)
		}
	}
}

func TestValidateURL(t *testing.T) {
	tests := []struct {
		input    []string
		expected bool
	}{
		{[]string{"http://example.com"}, true},
		{[]string{"https://example.com", "http://another.com"}, true},
		{[]string{"example.com"}, false},
		{[]string{"http://example.com", "not_a_valid_url"}, false},
	}

	for _, test := range tests {
		if result := validateURL(test.input); result != test.expected {
			t.Errorf("for input %v, expected %v, got %v", test.input, test.expected, result)
		}
	}
}
