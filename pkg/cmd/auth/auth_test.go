package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	coreauth "github.com/coreeng/corectl/pkg/auth"
	"github.com/coreeng/corectl/pkg/cmd/platformruntime"
	"github.com/coreeng/corectl/pkg/instance"
	"github.com/coreeng/corectl/pkg/portal"
	"github.com/stretchr/testify/require"
)

type memoryTokens struct{ tokens map[string]coreauth.Token }

func (m *memoryTokens) Get(origin string) (coreauth.Token, error) { return m.tokens[origin], nil }
func (m *memoryTokens) Set(origin string, token coreauth.Token) error {
	m.tokens[origin] = token
	return nil
}
func (m *memoryTokens) Delete(origin string) error { delete(m.tokens, origin); return nil }

func TestLoginUsesDiscoveryAndStoresTokenForSelectedOrigin(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case portal.DiscoveryPath:
			_ = json.NewEncoder(w).Encode(portal.Discovery{
				ClientID:                    "corectl",
				Scopes:                      "openid profile",
				DeviceAuthorizationEndpoint: server.URL + portal.DeviceAuthorizationPath,
				TokenEndpoint:               server.URL + portal.DeviceTokenPath,
			})
		case portal.DeviceAuthorizationPath:
			_ = json.NewEncoder(w).Encode(portal.DeviceAuthorization{DeviceCode: "device", UserCode: "ABCD", VerificationURI: server.URL + "/activate", ExpiresIn: 60})
		case portal.DeviceTokenPath:
			_ = json.NewEncoder(w).Encode(portal.DeviceToken{AccessToken: "access", TokenType: "Bearer", ExpiresIn: 3600})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	instances := &instance.Store{Path: filepath.Join(t.TempDir(), "platform.json")}
	require.NoError(t, instances.Add("test", server.URL))
	tokens := &memoryTokens{tokens: map[string]coreauth.Token{}}
	var opened string
	runtime := &platformruntime.Runtime{
		Instances:    instances,
		Tokens:       tokens,
		HTTPClient:   server.Client(),
		InstanceName: "test",
		OpenURL: func(url string) error {
			opened = url
			return nil
		},
		Sleep: func(context.Context, time.Duration) error { return nil },
	}
	output := new(bytes.Buffer)
	cmd := LoginCmd(runtime)
	cmd.SetOut(output)

	require.NoError(t, cmd.Execute())
	require.Equal(t, server.URL+"/activate", opened)
	require.Equal(t, "access", tokens.tokens[server.URL].AccessToken)
	require.Contains(t, output.String(), "Logged in to test")
}
