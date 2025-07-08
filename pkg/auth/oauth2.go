package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/logger"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// IDTokenClaims represents the claims in an ID token
type IDTokenClaims struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	Iss           string `json:"iss"`
	Aud           string `json:"aud"`
	Exp           int64  `json:"exp"`
	Iat           int64  `json:"iat"`
	EmailVerified bool   `json:"email_verified"`
}

// GetIDToken retrieves the stored ID token from config, automatically refreshing if expired
func GetIDToken(cfg *config.Config) (string, error) {
	idToken := cfg.OAuth2.IDToken.Value
	if idToken == "" {
		return "", fmt.Errorf("no ID token found. Please run 'corectl auth login' to authenticate")
	}

	// Check if token is expired and try to refresh
	if isTokenExpired(cfg) {
		// Try to refresh the token automatically
		clientID, clientSecret, err := getOAuthCredentials()
		if err != nil {
			logger.Debug().Msgf("Could not get OAuth credentials for automatic refresh: %v", err)
			return "", fmt.Errorf("ID token has expired. Please run 'corectl auth login' to re-authenticate")
		}

		logger.Debug().Msg("ID token expired, attempting automatic refresh")
		if err := RefreshToken(cfg, clientID, clientSecret); err != nil {
			logger.Debug().Msgf("Automatic token refresh failed: %v", err)
			return "", fmt.Errorf("ID token has expired and refresh failed. Please run 'corectl auth login' to re-authenticate")
		}

		// Get the refreshed token
		idToken = cfg.OAuth2.IDToken.Value
		if idToken == "" {
			return "", fmt.Errorf("token refresh succeeded but no ID token received. Please run 'corectl auth login' to re-authenticate")
		}

		logger.Debug().Msg("Token automatically refreshed successfully")
	}

	return idToken, nil
}

// IsAuthenticated checks if the user is authenticated with a valid token, attempting refresh if expired
func IsAuthenticated(cfg *config.Config) bool {
	if cfg.OAuth2.IDToken.Value == "" {
		return false
	}

	// If token is not expired, we're authenticated
	if !isTokenExpired(cfg) {
		return true
	}

	// Try to refresh the token automatically
	clientID, clientSecret, err := getOAuthCredentials()
	if err != nil {
		logger.Debug().Msgf("Could not get OAuth credentials for automatic refresh: %v", err)
		return false
	}

	logger.Debug().Msg("ID token expired, attempting automatic refresh for authentication check")
	if err := RefreshToken(cfg, clientID, clientSecret); err != nil {
		logger.Debug().Msgf("Automatic token refresh failed: %v", err)
		return false
	}

	logger.Debug().Msg("Token automatically refreshed successfully")
	return true
}

// isTokenExpired checks if the stored token is expired
func isTokenExpired(cfg *config.Config) bool {
	expiryStr := cfg.OAuth2.TokenExpiry.Value
	if expiryStr == "" {
		return true
	}

	expiry, err := time.Parse(time.RFC3339, expiryStr)
	if err != nil {
		logger.Warn().Msgf("Invalid token expiry format: %v", err)
		return true
	}

	// Add a small buffer to account for clock skew
	return time.Now().Add(30 * time.Second).After(expiry)
}

// GetUserInfo retrieves user information from the ID token
func GetUserInfo(cfg *config.Config) (*IDTokenClaims, error) {
	idToken, err := GetIDToken(cfg)
	if err != nil {
		return nil, err
	}

	// Parse the ID token to extract claims
	// Note: In a production environment, you should properly validate the JWT signature
	claims, err := parseIDTokenClaims(idToken)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ID token: %w", err)
	}

	return claims, nil
}

// parseIDTokenClaims parses the ID token to extract claims
// This is a simplified version - in production, you should use a proper JWT library
func parseIDTokenClaims(idToken string) (*IDTokenClaims, error) {
	// This is a simplified implementation
	// In production, you should use a proper JWT library like github.com/golang-jwt/jwt

	// For now, we'll make a request to Google's tokeninfo endpoint
	url := fmt.Sprintf("https://oauth2.googleapis.com/tokeninfo?id_token=%s", idToken)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to validate ID token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ID token validation failed with status: %d", resp.StatusCode)
	}

	var claims IDTokenClaims
	if err := json.NewDecoder(resp.Body).Decode(&claims); err != nil {
		return nil, fmt.Errorf("failed to decode token claims: %w", err)
	}

	return &claims, nil
}

// RefreshToken attempts to refresh the access token using the refresh token
func RefreshToken(cfg *config.Config, clientID, clientSecret string) error {
	refreshToken := cfg.OAuth2.RefreshToken.Value
	if refreshToken == "" {
		return fmt.Errorf("no refresh token available. Please run 'corectl auth login' to re-authenticate")
	}

	oauthConfig := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     google.Endpoint,
	}

	token := &oauth2.Token{
		RefreshToken: refreshToken,
	}

	tokenSource := oauthConfig.TokenSource(context.Background(), token)

	newToken, err := tokenSource.Token()
	if err != nil {
		return fmt.Errorf("failed to refresh token: %w", err)
	}

	// Update stored tokens
	cfg.OAuth2.AccessToken.Value = newToken.AccessToken
	cfg.OAuth2.TokenExpiry.Value = newToken.Expiry.Format(time.RFC3339)

	if idToken, ok := newToken.Extra("id_token").(string); ok {
		cfg.OAuth2.IDToken.Value = idToken
	}

	if newToken.RefreshToken != "" {
		cfg.OAuth2.RefreshToken.Value = newToken.RefreshToken
	}

	return cfg.Save()
}

// CreateIAPTransport creates an HTTP transport that adds the ID token to requests
func CreateIAPTransport(cfg *config.Config) (http.RoundTripper, error) {
	idToken, err := GetIDToken(cfg)
	if err != nil {
		return nil, err
	}

	return &IAPTransport{
		IDToken: idToken,
		Base:    http.DefaultTransport,
	}, nil
}

// IAPTransport is an HTTP transport that adds the ID token to requests for IAP
type IAPTransport struct {
	IDToken string
	Base    http.RoundTripper
}

func (t *IAPTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone the request to avoid modifying the original
	newReq := req.Clone(req.Context())

	// Add the ID token as a Bearer token in the Authorization header
	newReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", t.IDToken))

	return t.Base.RoundTrip(newReq)
}

// Logout clears the stored authentication tokens
func Logout(cfg *config.Config) error {
	cfg.OAuth2.IDToken.Value = ""
	cfg.OAuth2.AccessToken.Value = ""
	cfg.OAuth2.RefreshToken.Value = ""
	cfg.OAuth2.TokenExpiry.Value = ""

	return cfg.Save()
}

// getOAuthCredentials attempts to get OAuth credentials from various sources
func getOAuthCredentials() (string, string, error) {
	// Try environment variables first
	if clientID := os.Getenv("OAUTH_CLIENT_ID"); clientID != "" {
		clientSecret := os.Getenv("OAUTH_CLIENT_SECRET")
		if clientSecret == "" {
			return "", "", fmt.Errorf("OAUTH_CLIENT_SECRET environment variable required when OAUTH_CLIENT_ID is set")
		}
		return clientID, clientSecret, nil
	}

	// Try to get from Google Application Default Credentials
	if creds, err := google.FindDefaultCredentials(context.Background()); err == nil {
		if creds.JSON != nil {
			var credStruct struct {
				ClientID     string `json:"client_id"`
				ClientSecret string `json:"client_secret"`
			}
			if err := json.Unmarshal(creds.JSON, &credStruct); err == nil {
				if credStruct.ClientID != "" && credStruct.ClientSecret != "" {
					return credStruct.ClientID, credStruct.ClientSecret, nil
				}
			}
		}
	}

	return "", "", fmt.Errorf("OAuth 2.0 client credentials not found. Please provide via OAUTH_CLIENT_ID and OAUTH_CLIENT_SECRET environment variables, or configure Google Application Default Credentials")
}
