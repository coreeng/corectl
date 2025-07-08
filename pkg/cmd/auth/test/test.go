package test

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/coreeng/corectl/pkg/auth"
	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/logger"
	"github.com/spf13/cobra"
)

const (
	defaultTestURL = "https://core-platform-dashboard-integration.gcp-dev.cecg.platform.cecg.io/api/status"
)

func NewTestCmd(cfg *config.Config) *cobra.Command {
	var url string
	var timeout time.Duration

	testCmd := &cobra.Command{
		Use:   "test",
		Short: "Test authentication with API endpoint",
		Long: `Test authentication by making a request to an API endpoint.
This command will use your stored authentication token to make a request
to the specified endpoint with the JWT in the Authorization Bearer header.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTest(cfg, url, timeout)
		},
	}

	testCmd.Flags().StringVarP(&url, "url", "u", defaultTestURL, "URL to test authentication against")
	testCmd.Flags().DurationVarP(&timeout, "timeout", "t", 30*time.Second, "Request timeout")

	return testCmd
}

func runTest(cfg *config.Config, url string, timeout time.Duration) error {
	// Check if user is authenticated
	if !auth.IsAuthenticated(cfg) {
		return fmt.Errorf("not authenticated. Please run 'corectl auth login' first")
	}

	// Get the ID token
	idToken, err := auth.GetIDToken(cfg)
	if err != nil {
		return fmt.Errorf("failed to get ID token: %w", err)
	}

	// Print the JWT for debugging
	//fmt.Printf("Using JWT: %s\n", idToken)
	//fmt.Printf("Testing endpoint: %s\n", url)
	//fmt.Println()

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: timeout,
	}

	// Create the request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add the JWT to the Authorization Bearer header
	req.Header.Set("Authorization", "Bearer "+idToken)
	req.Header.Set("User-Agent", "corectl-auth-test")

	// Make the request
	fmt.Printf("Making request to %s...\n", url)
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Print the results
	fmt.Printf("Response Status: %s\n", resp.Status)
	fmt.Printf("Response Body:\n%s\n", string(body))

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		fmt.Printf("\nAuthentication test successful!\n")

		// Try to get user info for additional context
		if userInfo, err := auth.GetUserInfo(cfg); err == nil {
			fmt.Printf("Authenticated as: %s (%s)\n", userInfo.Name, userInfo.Email)
		}
	} else {
		fmt.Printf("\nAuthentication test failed with status: %s\n", resp.Status)

		// Log additional debugging info
		logger.Debug().Msgf("Auth test failed: url=%s, status_code=%d, response_body=%s", url, resp.StatusCode, string(body))
	}

	return nil
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}
