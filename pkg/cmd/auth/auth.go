package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	coreauth "github.com/coreeng/corectl/pkg/auth"
	"github.com/coreeng/corectl/pkg/cmd/platformruntime"
	"github.com/coreeng/corectl/pkg/portal"
	"github.com/spf13/cobra"
)

func LoginCmd(runtime *platformruntime.Runtime) *cobra.Command {
	return &cobra.Command{Use: "login", Short: "Log in to a Core Platform instance", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		selected, err := runtime.Selected()
		if err != nil {
			return err
		}
		client := portal.New(selected.Origin, runtime.HTTPClient, "")
		discovery, err := client.Discovery(cmd.Context())
		if err != nil {
			return fmt.Errorf("discover Portal CLI endpoints: %w", err)
		}
		if discovery.ClientID == "" {
			return fmt.Errorf("Portal CLI discovery did not provide client_id")
		}
		authorization, err := client.AuthorizeDevice(cmd.Context(), discovery.DeviceAuthorizationEndpoint, discovery)
		if err != nil {
			return fmt.Errorf("start device authorization: %w", err)
		}
		if authorization.DeviceCode == "" || authorization.UserCode == "" || authorization.VerificationURI == "" || authorization.ExpiresIn <= 0 {
			return fmt.Errorf("Portal returned an incomplete device authorization")
		}
		verificationURL := authorization.VerificationURIComplete
		if verificationURL == "" {
			verificationURL = authorization.VerificationURI
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Open %s and enter code %s\n", authorization.VerificationURI, authorization.UserCode)
		if verificationURL != "" && runtime.OpenURL != nil {
			_ = runtime.OpenURL(verificationURL)
		}
		interval := time.Duration(authorization.Interval) * time.Second
		if interval <= 0 {
			interval = 5 * time.Second
		}
		ctx, cancel := context.WithTimeout(cmd.Context(), time.Duration(authorization.ExpiresIn)*time.Second)
		defer cancel()
		for {
			if err := runtime.Sleep(ctx, interval); err != nil {
				return fmt.Errorf("device authorization expired or was cancelled: %w", err)
			}
			token, err := client.PollDeviceToken(ctx, discovery.TokenEndpoint, discovery, authorization.DeviceCode)
			if err != nil {
				var apiErr *portal.APIError
				if errors.As(err, &apiErr) && apiErr.Code == "authorization_pending" {
					continue
				}
				if errors.As(err, &apiErr) && apiErr.Code == "slow_down" {
					interval += 5 * time.Second
					continue
				}
				return fmt.Errorf("complete device authorization: %w", err)
			}
			if token.AccessToken == "" {
				return fmt.Errorf("Portal device token response did not include an access token")
			}
			expiresAt := time.Time{}
			if token.ExpiresIn > 0 {
				expiresAt = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
			}
			if err := runtime.Tokens.Set(selected.Origin, coreauth.Token{AccessToken: token.AccessToken, RefreshToken: token.RefreshToken, TokenType: token.TokenType, ExpiresAt: expiresAt}); err != nil {
				return fmt.Errorf("store token in OS keychain: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Logged in to %s\n", selected.Name)
			return nil
		}
	}}
}

func LogoutCmd(runtime *platformruntime.Runtime) *cobra.Command {
	return &cobra.Command{Use: "logout", Short: "Log out of a Core Platform instance", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		selected, err := runtime.Selected()
		if err != nil {
			return err
		}
		var remoteErr error
		if token, tokenErr := runtime.Tokens.Get(selected.Origin); tokenErr == nil && token.AccessToken != "" {
			remoteErr = portal.New(selected.Origin, runtime.HTTPClient, token.AccessToken).Logout(cmd.Context())
		}
		if err := runtime.Tokens.Delete(selected.Origin); err != nil {
			return fmt.Errorf("delete token from OS keychain: %w", err)
		}
		if remoteErr != nil {
			return fmt.Errorf("Portal logout failed after removing the local credential: %w", remoteErr)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Logged out of %s\n", selected.Name)
		return nil
	}}
}

func WhoamiCmd(runtime *platformruntime.Runtime) *cobra.Command {
	return &cobra.Command{Use: "whoami", Short: "Show the current Portal user", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		client, selected, err := runtime.AuthenticatedClient()
		if err != nil {
			return err
		}
		discovery, err := client.Discovery(cmd.Context())
		if err != nil {
			return err
		}
		user, err := client.User(cmd.Context(), discovery.UserinfoEndpoint)
		if err != nil {
			return fmt.Errorf("get current user: %w", err)
		}
		identity := user.Email
		if identity == "" {
			identity = user.Name
		}
		if identity == "" {
			identity = user.ID
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", identity, selected.Name)
		return nil
	}}
}
