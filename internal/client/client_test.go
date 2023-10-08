package client

import (
	"context"
	"errors"
	"net/http"
	"os"
	"site_monitoring/internal/model"
	"site_monitoring/sitemon/config"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// 1. Mock Dependencies
type MockHTTPClient struct{}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	// Simulate success or failure based on the request URL or headers
	if req.URL.String() == "http://fail.com" {
		return nil, errors.New("failed request")
	}
	if req.URL.String() == "http://failed.com" {
		return &http.Response{StatusCode: http.StatusInternalServerError}, nil
	}
	return &http.Response{StatusCode: http.StatusOK}, nil
}

type MockNotificator struct{}

func (m *MockNotificator) SendMessage(message string) error {
	// Capture or log the message for assertions
	return nil
}

func TestGetHeaders(t *testing.T) {
	configContent := `
timing:
  delay: 1s
`
	configFile, err := os.CreateTemp("", "config_*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(configFile.Name())

	if _, err := configFile.WriteString(configContent); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(configFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	log := logrus.New()
	notificator := &MockNotificator{}

	c := NewClient(log, notificator, cfg)
	c.C = &MockHTTPClient{}

	wg := &sync.WaitGroup{}
	wg.Add(1)
	ctx, cancel := context.WithCancel(context.Background())
	err = c.GetHeaders(ctx, "http://test.com", wg)
	time.Sleep(1 * time.Second)
	cancel()
	assert.Nil(t, err)
	wg.Wait()
}

func TestCheckRespStatus(t *testing.T) {
	// Given
	log := logrus.New()
	notificator := &MockNotificator{}
	cfg := &config.Config{} // Initialize with necessary values

	c := NewClient(log, notificator, cfg)

	testURL := "http://test.com"
	missingURL := "http://missing.com"
	errorURL := "http://error.com"

	// Setting up mock data
	m[testURL] = &model.Resp{ResponseCode: http.StatusOK}
	m[errorURL] = &model.Resp{ResponseCode: http.StatusInternalServerError}

	tests := []struct {
		url      string
		expected bool
		hasError bool
	}{
		{url: testURL, expected: true, hasError: false},
		{url: missingURL, expected: false, hasError: true},
		{url: errorURL, expected: false, hasError: false},
	}

	for _, tt := range tests {
		// When
		ok, err := c.CheckRespStatus(tt.url)

		// Then
		if tt.hasError {
			if err == nil {
				t.Errorf("Expected error for url %s, but got none", tt.url)
			}
		} else {
			if err != nil {
				t.Errorf("Unexpected error for url %s: %s", tt.url, err.Error())
			}
		}

		if ok != tt.expected {
			t.Errorf("For url %s expected %v but got %v", tt.url, tt.expected, ok)
		}
	}
}

func TestCheckRespContentLength(t *testing.T) {
	// 4. Test CheckRespContentLength method
	// ... Implement the test
}

func TestDo(t *testing.T) {
	// Given
	log := logrus.New()
	notificator := &MockNotificator{}
	cfg := &config.Config{} // Initialize with necessary values

	c := NewClient(log, notificator, cfg)
	c.C = &MockHTTPClient{} // Override the default client with our mock

	// Define some test scenarios
	tests := []struct {
		url              string
		mockResponseCode int
		expectedCode     int
		expectedErrCount uint
	}{
		{"http://success.com", http.StatusOK, http.StatusOK, 0},
		{"http://failed.com", http.StatusInternalServerError, http.StatusInternalServerError, 1},
		// Add more test cases as needed
	}

	for _, tt := range tests {
		// Create a mock response for the MockHTTPClient
		m[tt.url] = &model.Resp{ResponseCode: tt.mockResponseCode}

		// Create a new request for the test case
		req, err := http.NewRequest("GET", tt.url, nil)
		if err != nil {
			t.Fatalf("Failed to create request: %s", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Call the Do method
		c.Do(ctx, cancel, tt.url, m[tt.url], req, time.Now())

		// Verify the state in the shared map
		resp, exists := m[tt.url]
		if !exists {
			t.Errorf("Expected entry for URL %s in map, but not found", tt.url)
			continue
		}

		if resp.ResponseCode != tt.expectedCode {
			t.Errorf("Expected response code %d for URL %s but got %d", tt.expectedCode, tt.url, resp.ResponseCode)
		}

		if resp.ErrorCount != tt.expectedErrCount {
			t.Errorf("Expected error count %d for URL %s but got %d", tt.expectedErrCount, tt.url, resp.ErrorCount)
		}
	}
}
