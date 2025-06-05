package git

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/google/go-github/v60/github"
	"github.com/stretchr/testify/assert"
)

func TestRetryGitHubAPI_Success(t *testing.T) {
	callCount := 0
	operation := func() (*github.Repository, *github.Response, error) {
		callCount++
		return &github.Repository{}, &github.Response{}, nil
	}

	result, resp, err := RetryGitHubAPI(operation, 3, 10*time.Millisecond)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, resp)
	assert.Equal(t, 1, callCount, "Should succeed on first attempt")
}

func TestRetryGitHubAPI_404Retry(t *testing.T) {
	callCount := 0
	operation := func() (*github.Repository, *github.Response, error) {
		callCount++
		if callCount < 3 {
			return nil, &github.Response{Response: &http.Response{StatusCode: 404}}, errors.New("404 Not Found")
		}
		return &github.Repository{}, &github.Response{}, nil
	}

	start := time.Now()
	result, resp, err := RetryGitHubAPI(operation, 3, 10*time.Millisecond)
	duration := time.Since(start)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, resp)
	assert.Equal(t, 3, callCount, "Should retry 404 errors")
	// Should have some delay due to retries (at least 10ms + 20ms = 30ms)
	assert.Greater(t, duration, 25*time.Millisecond, "Should have retry delays")
}

func TestRetryGitHubAPI_NonRetryableError(t *testing.T) {
	callCount := 0
	operation := func() (*github.Repository, *github.Response, error) {
		callCount++
		return nil, &github.Response{Response: &http.Response{StatusCode: 500}}, errors.New("500 Internal Server Error")
	}

	result, resp, err := RetryGitHubAPI(operation, 3, 10*time.Millisecond)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.NotNil(t, resp)
	assert.Equal(t, 1, callCount, "Should not retry non-404 errors")
}

func TestRetryGitHubAPI_MaxRetriesExceeded(t *testing.T) {
	callCount := 0
	operation := func() (*github.Repository, *github.Response, error) {
		callCount++
		return nil, &github.Response{Response: &http.Response{StatusCode: 404}}, errors.New("404 Not Found")
	}

	result, resp, err := RetryGitHubAPI(operation, 2, 10*time.Millisecond)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.NotNil(t, resp)
	assert.Equal(t, 3, callCount, "Should try maxRetries+1 times")
}

func TestRetryGitHubAPI_403Retry(t *testing.T) {
	callCount := 0
	operation := func() (*github.Repository, *github.Response, error) {
		callCount++
		if callCount <= 2 {
			return nil, &github.Response{Response: &http.Response{StatusCode: 403}}, errors.New("403 Repository access blocked")
		}
		return &github.Repository{}, &github.Response{}, nil
	}

	result, resp, err := RetryGitHubAPI(operation, 3, 10*time.Millisecond)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, resp)
	assert.Equal(t, 3, callCount, "Should retry 403 errors in repository operations")
}

func TestIsGitHub404Error(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"404 error", errors.New("404 Not Found"), true},
		{"not found error", errors.New("repository not found"), true},
		{"Not Found error", errors.New("GET https://api.github.com/repos/org/repo: 404 Not Found []"), true},
		{"500 error", errors.New("500 Internal Server Error"), false},
		{"other error", errors.New("some other error"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsGitHub404Error(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRetryGitHubOperation_Success(t *testing.T) {
	callCount := 0
	operation := func() error {
		callCount++
		return nil
	}

	err := RetryGitHubOperation(operation, 3, 10*time.Millisecond)

	assert.NoError(t, err)
	assert.Equal(t, 1, callCount, "Should succeed on first attempt")
}

func TestRetryGitHubOperation_404Retry(t *testing.T) {
	callCount := 0
	operation := func() error {
		callCount++
		if callCount < 3 {
			return errors.New("404 Not Found")
		}
		return nil
	}

	start := time.Now()
	err := RetryGitHubOperation(operation, 3, 10*time.Millisecond)
	duration := time.Since(start)

	assert.NoError(t, err)
	assert.Equal(t, 3, callCount, "Should retry 404 errors")
	// Should have some delay due to retries (at least 10ms + 20ms = 30ms)
	assert.Greater(t, duration, 25*time.Millisecond, "Should have retry delays")
}

func TestRetryGitHubOperation_403Retry(t *testing.T) {
	callCount := 0
	operation := func() error {
		callCount++
		if callCount <= 2 {
			return errors.New("403 Repository access blocked")
		}
		return nil
	}

	err := RetryGitHubOperation(operation, 4, 10*time.Millisecond)

	assert.NoError(t, err)
	assert.Equal(t, 3, callCount, "Should retry 403 errors in repository operations context")
}

func TestIsGitHubPropagationError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"404 error", errors.New("404 Not Found"), true},
		{"repository not found", errors.New("repository not found"), true},
		{"403 Repository access blocked", errors.New("403 Repository access blocked"), true},
		{"403 Forbidden", errors.New("403 Forbidden"), true},
		{"access blocked", errors.New("DELETE https://api.github.com/repos/org/repo: 403 Repository access blocked []"), true},
		{"500 error", errors.New("500 Internal Server Error"), false},
		{"other error", errors.New("some other error"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsGitHubPropagationError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
