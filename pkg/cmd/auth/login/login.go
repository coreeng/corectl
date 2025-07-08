package login

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/coreeng/corectl/pkg/cmdutil/config"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/coreeng/corectl/pkg/logger"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const (
	// OAuth 2.0 scopes for IAP
	// https://developers.google.com/identity/protocols/oauth2/scopes
	defaultScope = "openid email profile"

	// Local server port for OAuth callback
	callbackPort = "8080"
	callbackPath = "/oauth/callback"

	// PKCE code verifier length
	codeVerifierLength = 128
)

type LoginOpt struct {
	ClientID       string
	ClientSecret   string
	NonInteractive bool
	Port           string
	Streams        userio.IOStreams
}

func NewLoginCmd(cfg *config.Config) *cobra.Command {
	opt := LoginOpt{
		Port: callbackPort,
	}

	loginCmd := &cobra.Command{
		Use:   "login",
		Short: "Login to authenticate with Google IAP using OAuth 2.0",
		Long: `Login to authenticate with Google IAP using OAuth 2.0 installed app flow.
		
This command will open a browser window for you to authenticate with Google,
then store the ID token for subsequent API calls behind IAP.

The OAuth 2.0 client credentials should be configured for a desktop application
and the callback URL should be set to http://localhost:8080/oauth/callback.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true

			nonInteractive, err := cmd.Flags().GetBool("non-interactive")
			if err != nil {
				return err
			}
			opt.NonInteractive = nonInteractive

			opt.Streams = userio.NewIOStreamsWithInteractive(
				cmd.InOrStdin(),
				cmd.OutOrStdout(),
				cmd.OutOrStderr(),
				!opt.NonInteractive,
			)

			return run(&opt, cfg)
		},
	}

	loginCmd.Flags().StringVar(
		&opt.ClientID,
		"client-id",
		"",
		"OAuth 2.0 client ID for desktop application",
	)

	loginCmd.Flags().StringVar(
		&opt.ClientSecret,
		"client-secret",
		"",
		"OAuth 2.0 client secret for desktop application",
	)

	loginCmd.Flags().StringVar(
		&opt.Port,
		"port",
		callbackPort,
		"Local port for OAuth callback server",
	)

	return loginCmd
}

func run(opt *LoginOpt, cfg *config.Config) error {
	ctx := context.Background()

	// Get OAuth 2.0 client credentials
	clientID, clientSecret, err := getOAuthCredentials(opt)
	if err != nil {
		return fmt.Errorf("failed to get OAuth credentials: %w", err)
	}

	// Configure OAuth 2.0
	oauthConfig := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes:       []string{defaultScope},
		Endpoint:     google.Endpoint,
		RedirectURL:  fmt.Sprintf("http://localhost:%s%s", opt.Port, callbackPath),
	}

	// Generate PKCE code verifier and challenge
	codeVerifier, err := generateCodeVerifier()
	if err != nil {
		return fmt.Errorf("failed to generate PKCE code verifier: %w", err)
	}

	codeChallenge := generateCodeChallenge(codeVerifier)

	// Start local callback server
	codeChan := make(chan string, 1)
	errorChan := make(chan error, 1)

	server := &http.Server{
		Addr: fmt.Sprintf(":%s", opt.Port),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == callbackPath {
				handleCallback(w, r, codeChan, errorChan)
			} else {
				http.NotFound(w, r)
			}
		}),
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errorChan <- fmt.Errorf("failed to start callback server: %w", err)
		}
	}()

	// Generate authorization URL with PKCE
	authURL := oauthConfig.AuthCodeURL("state",
		oauth2.SetAuthURLParam("code_challenge", codeChallenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)

	// Open browser
	opt.Streams.CurrentHandler = opt.Streams.Wizard("Opening browser for authentication", "Authentication completed")
	defer opt.Streams.CurrentHandler.Done()

	if !opt.NonInteractive {
		if err := openBrowser(authURL); err != nil {
			logger.Warn().Msgf("Could not open browser automatically: %v", err)
		}
	}

	opt.Streams.CurrentHandler.Info(fmt.Sprintf("Please open the following URL in your browser if it didn't open automatically:"))
	opt.Streams.CurrentHandler.Info(authURL)

	// Wait for callback
	var authCode string
	select {
	case authCode = <-codeChan:
		opt.Streams.CurrentHandler.Info("Authorization code received")
	case err := <-errorChan:
		return fmt.Errorf("OAuth callback error: %w", err)
	case <-time.After(5 * time.Minute):
		return fmt.Errorf("authentication timed out after 5 minutes")
	}

	// Shutdown callback server
	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Warn().Msgf("Error shutting down callback server: %v", err)
	}

	// Exchange authorization code for tokens
	token, err := oauthConfig.Exchange(ctx, authCode,
		oauth2.SetAuthURLParam("code_verifier", codeVerifier),
	)
	if err != nil {
		return fmt.Errorf("failed to exchange authorization code for token: %w", err)
	}

	// Store tokens in config
	if err := storeTokens(cfg, token); err != nil {
		return fmt.Errorf("failed to store tokens: %w", err)
	}

	opt.Streams.CurrentHandler.Info("Successfully logged in and stored authentication tokens")

	return nil
}

func getOAuthCredentials(opt *LoginOpt) (string, string, error) {
	// Try environment variables first
	if clientID := os.Getenv("OAUTH_CLIENT_ID"); clientID != "" {
		clientSecret := os.Getenv("OAUTH_CLIENT_SECRET")
		if clientSecret == "" {
			return "", "", fmt.Errorf("OAUTH_CLIENT_SECRET environment variable required when OAUTH_CLIENT_ID is set")
		}
		return clientID, clientSecret, nil
	}

	// Use command line flags
	if opt.ClientID != "" {
		if opt.ClientSecret == "" {
			return "", "", fmt.Errorf("--client-secret required when --client-id is provided")
		}
		return opt.ClientID, opt.ClientSecret, nil
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

	return "", "", fmt.Errorf("OAuth 2.0 client credentials not found. Please provide via --client-id and --client-secret flags, or set OAUTH_CLIENT_ID and OAUTH_CLIENT_SECRET environment variables")
}

func generateCodeVerifier() (string, error) {
	bytes := make([]byte, codeVerifierLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(bytes), nil
}

func generateCodeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(hash[:])
}

func handleCallback(w http.ResponseWriter, r *http.Request, codeChan chan string, errorChan chan error) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query()

	// Check for error
	if errParam := query.Get("error"); errParam != "" {
		errorMsg := fmt.Sprintf("OAuth error: %s", errParam)
		if desc := query.Get("error_description"); desc != "" {
			errorMsg += fmt.Sprintf(" (%s)", desc)
		}

		http.Error(w, errorMsg, http.StatusBadRequest)
		errorChan <- fmt.Errorf(errorMsg)
		return
	}

	// Get authorization code
	code := query.Get("code")
	if code == "" {
		http.Error(w, "Missing authorization code", http.StatusBadRequest)
		errorChan <- fmt.Errorf("missing authorization code in callback")
		return
	}

	// Send success response
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `
		<html>
		<body>
			<h1>Authentication Successful</h1>
			<p>You have successfully authenticated with Google IAP. You can now close this window.</p>
			<script>window.close();</script>
		</body>
		</html>
	`)

	codeChan <- code
}

func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "linux":
		cmd = "xdg-open"
		args = []string{url}
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	return exec.Command(cmd, args...).Start()
}

func storeTokens(cfg *config.Config, token *oauth2.Token) error {
	// Store the ID token in config
	if idToken, ok := token.Extra("id_token").(string); ok {
		cfg.OAuth2.IDToken.Value = idToken
	}

	// Store refresh token if available
	if token.RefreshToken != "" {
		cfg.OAuth2.RefreshToken.Value = token.RefreshToken
	}

	// Store access token and expiry
	cfg.OAuth2.AccessToken.Value = token.AccessToken
	cfg.OAuth2.TokenExpiry.Value = token.Expiry.Format(time.RFC3339)

	return cfg.Save()
}
