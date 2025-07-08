package test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/stretchr/testify/assert"
)

func TestNewTestCmd(t *testing.T) {
	cfg := &config.Config{}
	cmd := NewTestCmd(cfg)

	assert.Equal(t, "test", cmd.Use)
	assert.Equal(t, "Test authentication with API endpoint", cmd.Short)
	assert.Contains(t, cmd.Long, "Test authentication by making a request to an API endpoint")

	// Check flags
	urlFlag := cmd.Flags().Lookup("url")
	assert.NotNil(t, urlFlag)
	assert.Equal(t, defaultTestURL, urlFlag.DefValue)

	timeoutFlag := cmd.Flags().Lookup("timeout")
	assert.NotNil(t, timeoutFlag)
	assert.Equal(t, "30s", timeoutFlag.DefValue)
}

func TestRunTest_NotAuthenticated(t *testing.T) {
	cfg := &config.Config{}

	err := runTest(cfg, "http://example.com", 30*time.Second)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not authenticated")
	assert.Contains(t, err.Error(), "corectl auth login")
}

func TestRunTest_Success(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the Authorization Bearer header is set
		authHeader := r.Header.Get("Authorization")
		assert.Equal(t, "Bearer test-jwt-token", authHeader)

		// Verify User-Agent header
		userAgent := r.Header.Get("User-Agent")
		assert.Equal(t, "corectl-auth-test", userAgent)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	// Create config with valid auth
	cfg := &config.Config{
		OAuth2: config.OAuth2Config{
			IDToken: config.Parameter[string]{
				Value: "test-jwt-token",
			},
			TokenExpiry: config.Parameter[string]{
				Value: time.Now().Add(1 * time.Hour).Format(time.RFC3339),
			},
		},
	}

	// Run the test
	err := runTest(cfg, server.URL, 30*time.Second)

	// Should not error
	assert.NoError(t, err)
}

func TestRunTest_Failure(t *testing.T) {
	// Create a test server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "unauthorized"}`))
	}))
	defer server.Close()

	// Create config with valid auth
	cfg := &config.Config{
		OAuth2: config.OAuth2Config{
			IDToken: config.Parameter[string]{
				Value: "test-jwt-token",
			},
			TokenExpiry: config.Parameter[string]{
				Value: time.Now().Add(1 * time.Hour).Format(time.RFC3339),
			},
		},
	}

	// Run the test
	err := runTest(cfg, server.URL, 30*time.Second)

	// Should not error, but the output should indicate failure
	assert.NoError(t, err)
}

func TestRunTest_Timeout(t *testing.T) {
	// Create a test server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	// Create config with valid auth
	cfg := &config.Config{
		OAuth2: config.OAuth2Config{
			IDToken: config.Parameter[string]{
				Value: "test-jwt-token",
			},
			TokenExpiry: config.Parameter[string]{
				Value: time.Now().Add(1 * time.Hour).Format(time.RFC3339),
			},
		},
	}

	// Run the test with very short timeout
	err := runTest(cfg, server.URL, 10*time.Millisecond)

	// Should error due to timeout
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "request failed")
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "string shorter than max",
			input:    "short",
			maxLen:   10,
			expected: "short",
		},
		{
			name:     "string equal to max",
			input:    "exactly10!",
			maxLen:   10,
			expected: "exactly10!",
		},
		{
			name:     "string longer than max",
			input:    "this is a very long string",
			maxLen:   10,
			expected: "this is a ",
		},
		{
			name:     "empty string",
			input:    "",
			maxLen:   10,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateString(tt.input, tt.maxLen)
			assert.Equal(t, tt.expected, result)
		})
	}
}
