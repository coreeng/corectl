package git

import (
	"net/http"
	"strings"
	"time"

	"github.com/coreeng/corectl/pkg/logger"
	"github.com/google/go-github/v60/github"
	"go.uber.org/zap"
)

// RetryGitHubAPI retries a GitHub API operation with exponential backoff.
// This is specifically designed to handle propagation delays after repository creation.
func RetryGitHubAPI[T any](operation func() (T, *github.Response, error), maxRetries int, baseDelay time.Duration) (T, *github.Response, error) {
	var result T
	var resp *github.Response
	var err error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		result, resp, err = operation()

		// If successful, return immediately
		if err == nil {
			if attempt > 0 {
				logger.Info().With(zap.Int("attempt", attempt+1)).Msg("github: API call succeeded after retry")
			}
			return result, resp, nil
		}

		// Check if this is a 404 error that might be due to propagation delay
		if resp != nil && resp.StatusCode == http.StatusNotFound && attempt < maxRetries {
			delay := baseDelay * time.Duration(1<<attempt) // Exponential backoff: 2s, 4s, 8s, 16s
			logger.Warn().With(
				zap.Int("attempt", attempt+1),
				zap.Int("max_retries", maxRetries+1),
				zap.Duration("delay", delay),
				zap.Int("status_code", resp.StatusCode),
			).Msg("github: API call failed with 404, retrying due to potential propagation delay")

			time.Sleep(delay)
			continue
		}

		// For non-404 errors or final attempt, return the error
		break
	}

	return result, resp, err
}

// IsGitHub404Error checks if an error is likely a GitHub 404 error
func IsGitHub404Error(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	// Check for common patterns in GitHub 404 errors
	return strings.Contains(errStr, "404") ||
		strings.Contains(errStr, "Not Found") ||
		strings.Contains(errStr, "not found")
}

// RetryGitHubOperation retries a GitHub operation that returns only an error with exponential backoff.
func RetryGitHubOperation(operation func() error, maxRetries int, baseDelay time.Duration) error {
	for attempt := 0; attempt <= maxRetries; attempt++ {
		err := operation()

		// If successful, return immediately
		if err == nil {
			if attempt > 0 {
				logger.Info().With(zap.Int("attempt", attempt+1)).Msg("github: operation succeeded after retry")
			}
			return nil
		}

		// Check if this is a 404-related error that might be due to propagation delay
		if IsGitHub404Error(err) && attempt < maxRetries {
			delay := baseDelay * time.Duration(1<<attempt) // Exponential backoff: 2s, 4s, 8s, 16s
			logger.Warn().With(
				zap.Int("attempt", attempt+1),
				zap.Int("max_retries", maxRetries+1),
				zap.Duration("delay", delay),
				zap.Error(err),
			).Msg("github: operation failed with 404-like error, retrying due to potential propagation delay")

			time.Sleep(delay)
			continue
		}

		// For non-404 errors or final attempt, return the error
		return err
	}

	return nil
}

// DefaultRetryConfig provides sensible defaults for GitHub API retries
const (
	DefaultMaxRetries = 4               // 5 total attempts (0-4)
	DefaultBaseDelay  = 2 * time.Second // Base delay of 2 seconds
)
