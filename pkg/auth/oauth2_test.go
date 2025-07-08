package auth

import (
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockConfig wraps config.Config for testing
type MockConfig struct {
	*config.Config
	SaveError error
}

func (mc *MockConfig) Save() error {
	return mc.SaveError
}

func TestIsAuthenticated(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *config.Config
		expected bool
	}{
		{
			name: "no token",
			cfg: &config.Config{
				OAuth2: config.OAuth2Config{
					IDToken: config.Parameter[string]{Value: ""},
				},
			},
			expected: false,
		},
		{
			name: "valid token",
			cfg: &config.Config{
				OAuth2: config.OAuth2Config{
					IDToken:     config.Parameter[string]{Value: "valid-token"},
					TokenExpiry: config.Parameter[string]{Value: time.Now().Add(time.Hour).Format(time.RFC3339)},
				},
			},
			expected: true,
		},
		{
			name: "expired token",
			cfg: &config.Config{
				OAuth2: config.OAuth2Config{
					IDToken:     config.Parameter[string]{Value: "expired-token"},
					TokenExpiry: config.Parameter[string]{Value: time.Now().Add(-time.Hour).Format(time.RFC3339)},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsAuthenticated(tt.cfg)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetIDToken(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *config.Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "no token",
			cfg: &config.Config{
				OAuth2: config.OAuth2Config{
					IDToken: config.Parameter[string]{Value: ""},
				},
			},
			expectError: true,
			errorMsg:    "no ID token found",
		},
		{
			name: "valid token",
			cfg: &config.Config{
				OAuth2: config.OAuth2Config{
					IDToken:     config.Parameter[string]{Value: "valid-token"},
					TokenExpiry: config.Parameter[string]{Value: time.Now().Add(time.Hour).Format(time.RFC3339)},
				},
			},
			expectError: false,
		},
		{
			name: "expired token",
			cfg: &config.Config{
				OAuth2: config.OAuth2Config{
					IDToken:     config.Parameter[string]{Value: "expired-token"},
					TokenExpiry: config.Parameter[string]{Value: time.Now().Add(-time.Hour).Format(time.RFC3339)},
				},
			},
			expectError: true,
			errorMsg:    "ID token has expired",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := GetIDToken(tt.cfg)
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.cfg.OAuth2.IDToken.Value, token)
			}
		})
	}
}

func TestIsTokenExpired(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *config.Config
		expected bool
	}{
		{
			name: "no expiry",
			cfg: &config.Config{
				OAuth2: config.OAuth2Config{
					TokenExpiry: config.Parameter[string]{Value: ""},
				},
			},
			expected: true,
		},
		{
			name: "invalid expiry format",
			cfg: &config.Config{
				OAuth2: config.OAuth2Config{
					TokenExpiry: config.Parameter[string]{Value: "invalid-format"},
				},
			},
			expected: true,
		},
		{
			name: "valid not expired",
			cfg: &config.Config{
				OAuth2: config.OAuth2Config{
					TokenExpiry: config.Parameter[string]{Value: time.Now().Add(time.Hour).Format(time.RFC3339)},
				},
			},
			expected: false,
		},
		{
			name: "expired",
			cfg: &config.Config{
				OAuth2: config.OAuth2Config{
					TokenExpiry: config.Parameter[string]{Value: time.Now().Add(-time.Hour).Format(time.RFC3339)},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isTokenExpired(tt.cfg)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLogout(t *testing.T) {
	// Create a mock config with tokens
	cfg := &MockConfig{
		Config: &config.Config{
			OAuth2: config.OAuth2Config{
				IDToken:      config.Parameter[string]{Value: "id-token"},
				AccessToken:  config.Parameter[string]{Value: "access-token"},
				RefreshToken: config.Parameter[string]{Value: "refresh-token"},
				TokenExpiry:  config.Parameter[string]{Value: time.Now().Add(time.Hour).Format(time.RFC3339)},
			},
		},
		SaveError: nil,
	}

	// Logout
	err := Logout(cfg.Config)
	require.NoError(t, err)

	// Verify all tokens are cleared
	assert.Empty(t, cfg.OAuth2.IDToken.Value)
	assert.Empty(t, cfg.OAuth2.AccessToken.Value)
	assert.Empty(t, cfg.OAuth2.RefreshToken.Value)
	assert.Empty(t, cfg.OAuth2.TokenExpiry.Value)
}

func TestCreateIAPTransport(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *config.Config
		expectError bool
	}{
		{
			name: "no token",
			cfg: &config.Config{
				OAuth2: config.OAuth2Config{
					IDToken: config.Parameter[string]{Value: ""},
				},
			},
			expectError: true,
		},
		{
			name: "valid token",
			cfg: &config.Config{
				OAuth2: config.OAuth2Config{
					IDToken:     config.Parameter[string]{Value: "valid-token"},
					TokenExpiry: config.Parameter[string]{Value: time.Now().Add(time.Hour).Format(time.RFC3339)},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport, err := CreateIAPTransport(tt.cfg)
			if tt.expectError {
				require.Error(t, err)
				assert.Nil(t, transport)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, transport)

				// Check that it's an IAPTransport
				iapTransport, ok := transport.(*IAPTransport)
				require.True(t, ok)
				assert.Equal(t, tt.cfg.OAuth2.IDToken.Value, iapTransport.IDToken)
			}
		})
	}
}

func TestGetOAuthCredentials(t *testing.T) {
	// Save original environment
	originalClientID := os.Getenv("OAUTH_CLIENT_ID")
	originalClientSecret := os.Getenv("OAUTH_CLIENT_SECRET")
	defer func() {
		os.Setenv("OAUTH_CLIENT_ID", originalClientID)
		os.Setenv("OAUTH_CLIENT_SECRET", originalClientSecret)
	}()

	tests := []struct {
		name           string
		setupEnv       func()
		expectError    bool
		expectedID     string
		expectedSecret string
	}{
		{
			name: "env vars set",
			setupEnv: func() {
				os.Setenv("OAUTH_CLIENT_ID", "test-client-id")
				os.Setenv("OAUTH_CLIENT_SECRET", "test-client-secret")
			},
			expectError:    false,
			expectedID:     "test-client-id",
			expectedSecret: "test-client-secret",
		},
		{
			name: "client ID without secret",
			setupEnv: func() {
				os.Setenv("OAUTH_CLIENT_ID", "test-client-id")
				os.Unsetenv("OAUTH_CLIENT_SECRET")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupEnv()

			clientID, clientSecret, err := getOAuthCredentials()
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedID, clientID)
				assert.Equal(t, tt.expectedSecret, clientSecret)
			}
		})
	}
}

func TestGetIDTokenWithAutomaticRefresh(t *testing.T) {
	// Save original environment
	originalClientID := os.Getenv("OAUTH_CLIENT_ID")
	originalClientSecret := os.Getenv("OAUTH_CLIENT_SECRET")
	defer func() {
		os.Setenv("OAUTH_CLIENT_ID", originalClientID)
		os.Setenv("OAUTH_CLIENT_SECRET", originalClientSecret)
	}()

	tests := []struct {
		name        string
		setupEnv    func()
		cfg         *config.Config
		setupServer func() *httptest.Server
		expectError bool
		errorMsg    string
	}{
		{
			name: "expired token, credentials available, refresh fails - should fail",
			setupEnv: func() {
				os.Setenv("OAUTH_CLIENT_ID", "test-client-id")
				os.Setenv("OAUTH_CLIENT_SECRET", "test-client-secret")
			},
			cfg: &config.Config{
				OAuth2: config.OAuth2Config{
					IDToken:      config.Parameter[string]{Value: "expired-token"},
					TokenExpiry:  config.Parameter[string]{Value: time.Now().Add(-time.Hour).Format(time.RFC3339)},
					RefreshToken: config.Parameter[string]{Value: "invalid-refresh-token"},
				},
			},
			expectError: true,
			errorMsg:    "ID token has expired and refresh failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupEnv()

			token, err := GetIDToken(tt.cfg)
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, token)
			}
		})
	}
}

func TestIsAuthenticatedWithAutomaticRefresh(t *testing.T) {
	// Save original environment
	originalClientID := os.Getenv("OAUTH_CLIENT_ID")
	originalClientSecret := os.Getenv("OAUTH_CLIENT_SECRET")
	defer func() {
		os.Setenv("OAUTH_CLIENT_ID", originalClientID)
		os.Setenv("OAUTH_CLIENT_SECRET", originalClientSecret)
	}()

	tests := []struct {
		name     string
		setupEnv func()
		cfg      *config.Config
		expected bool
	}{
		{
			name: "expired token, credentials available, invalid refresh token - should return false",
			setupEnv: func() {
				os.Setenv("OAUTH_CLIENT_ID", "test-client-id")
				os.Setenv("OAUTH_CLIENT_SECRET", "test-client-secret")
			},
			cfg: &config.Config{
				OAuth2: config.OAuth2Config{
					IDToken:      config.Parameter[string]{Value: "expired-token"},
					TokenExpiry:  config.Parameter[string]{Value: time.Now().Add(-time.Hour).Format(time.RFC3339)},
					RefreshToken: config.Parameter[string]{Value: "invalid-refresh-token"},
				},
			},
			expected: false,
		},
		{
			name: "valid token - should return true",
			setupEnv: func() {
				os.Unsetenv("OAUTH_CLIENT_ID")
				os.Unsetenv("OAUTH_CLIENT_SECRET")
			},
			cfg: &config.Config{
				OAuth2: config.OAuth2Config{
					IDToken:     config.Parameter[string]{Value: "valid-token"},
					TokenExpiry: config.Parameter[string]{Value: time.Now().Add(time.Hour).Format(time.RFC3339)},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupEnv()

			result := IsAuthenticated(tt.cfg)
			assert.Equal(t, tt.expected, result)
		})
	}
}
