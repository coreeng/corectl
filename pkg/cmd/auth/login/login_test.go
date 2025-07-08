package login

import (
	"os"
	"testing"

	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLoginCmd(t *testing.T) {
	cfg := config.NewConfig()
	cmd := NewLoginCmd(cfg)

	assert.Equal(t, "login", cmd.Use)
	assert.Equal(t, "Login to authenticate with Google IAP using OAuth 2.0", cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	assert.NotNil(t, cmd.RunE)

	// Check flags
	clientIDFlag := cmd.Flags().Lookup("client-id")
	assert.NotNil(t, clientIDFlag)
	assert.Equal(t, "", clientIDFlag.DefValue)

	clientSecretFlag := cmd.Flags().Lookup("client-secret")
	assert.NotNil(t, clientSecretFlag)
	assert.Equal(t, "", clientSecretFlag.DefValue)

	portFlag := cmd.Flags().Lookup("port")
	assert.NotNil(t, portFlag)
	assert.Equal(t, callbackPort, portFlag.DefValue)
}

func TestGetOAuthCredentials(t *testing.T) {
	tests := []struct {
		name        string
		opt         *LoginOpt
		envVars     map[string]string
		expectError bool
		errorMsg    string
	}{
		{
			name: "no credentials",
			opt: &LoginOpt{
				ClientID:     "",
				ClientSecret: "",
			},
			envVars:     map[string]string{},
			expectError: false, // May succeed if ADC is available on the system
			errorMsg:    "",
		},
		{
			name: "env vars - client ID only",
			opt: &LoginOpt{
				ClientID:     "",
				ClientSecret: "",
			},
			envVars: map[string]string{
				"OAUTH_CLIENT_ID": "test-client-id",
			},
			expectError: true,
			errorMsg:    "OAUTH_CLIENT_SECRET environment variable required",
		},
		{
			name: "env vars - both provided",
			opt: &LoginOpt{
				ClientID:     "",
				ClientSecret: "",
			},
			envVars: map[string]string{
				"OAUTH_CLIENT_ID":     "test-client-id",
				"OAUTH_CLIENT_SECRET": "test-client-secret",
			},
			expectError: false,
		},
		{
			name: "flags - client ID only",
			opt: &LoginOpt{
				ClientID:     "test-client-id",
				ClientSecret: "",
			},
			envVars:     map[string]string{},
			expectError: true,
			errorMsg:    "--client-secret required when --client-id is provided",
		},
		{
			name: "flags - both provided",
			opt: &LoginOpt{
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
			},
			envVars:     map[string]string{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}
			defer func() {
				for key := range tt.envVars {
					os.Unsetenv(key)
				}
			}()

			clientID, clientSecret, err := getOAuthCredentials(tt.opt)
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
				// Only check for non-empty credentials if we expect success
				if tt.name != "no credentials" {
					assert.NotEmpty(t, clientID)
					assert.NotEmpty(t, clientSecret)
				}
			}
		})
	}
}

func TestGenerateCodeVerifier(t *testing.T) {
	verifier, err := generateCodeVerifier()
	require.NoError(t, err)
	assert.NotEmpty(t, verifier)
	assert.True(t, len(verifier) > 0)
}

func TestGenerateCodeChallenge(t *testing.T) {
	verifier := "test-verifier"
	challenge := generateCodeChallenge(verifier)
	assert.NotEmpty(t, challenge)
	assert.True(t, len(challenge) > 0)

	// Same verifier should produce same challenge
	challenge2 := generateCodeChallenge(verifier)
	assert.Equal(t, challenge, challenge2)
}
