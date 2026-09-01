package portal

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (
	DiscoveryPath           = "/api/corectl/discovery"
	DeviceAuthorizationPath = "/api/auth/device/code"
	DeviceTokenPath         = "/api/auth/device/token"
	ClustersPath            = "/api/clusters"
	UserPath                = "/api/user/me"
	LogoutPath              = "/api/user/logout"
)

type Discovery struct {
	DeviceAuthorizationEndpoint string `json:"device_authorization_endpoint"`
	TokenEndpoint               string `json:"token_endpoint"`
	UserinfoEndpoint            string `json:"userinfo_endpoint"`
	ClientID                    string `json:"client_id"`
	Scopes                      string `json:"scopes"`
}

type DeviceAuthorization struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

type DeviceToken struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

// Generation accepts either a JSON string or number while keeping the value
// opaque to Corectl. Portal defines when a cluster binding becomes stale.
type Generation string

func (g *Generation) UnmarshalJSON(data []byte) error {
	var text string
	if len(data) > 0 && data[0] == '"' {
		if err := json.Unmarshal(data, &text); err != nil {
			return err
		}
	} else {
		text = string(data)
		if _, err := strconv.ParseInt(text, 10, 64); err != nil {
			return fmt.Errorf("invalid cluster generation %q", text)
		}
	}
	*g = Generation(text)
	return nil
}

type Cluster struct {
	ID                     string     `json:"id"`
	Name                   string     `json:"displayName"`
	Status                 string     `json:"lifecycle"`
	Generation             Generation `json:"generation"`
	KubeSystemNamespaceUID string     `json:"clusterFingerprint"`
}

type Chart struct {
	Reference string `json:"reference"`
	Version   string `json:"version"`
	Digest    string `json:"digest"`
}

type APIMetadata struct {
	BaseURL string `json:"baseUrl"`
}

type ReleaseMetadata struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Version   string `json:"version"`
}

type EnrollmentMetadata struct {
	Role      string `json:"role"`
	Token     string `json:"token"`
	ExpiresAt string `json:"expiresAt"`
	AttemptID string `json:"attemptId"`
}

type InstallationPlan struct {
	Operation                   string              `json:"operation"`
	ClusterID                   string              `json:"clusterId"`
	Generation                  Generation          `json:"generation"`
	ClusterFingerprint          string              `json:"clusterFingerprint"`
	API                         APIMetadata         `json:"api"`
	Release                     ReleaseMetadata     `json:"release"`
	Chart                       Chart               `json:"chart"`
	Enrollment                  *EnrollmentMetadata `json:"enrollment"`
	ControlEnrollment           *EnrollmentMetadata `json:"controlEnrollment"`
	ManagementBootstrapRequired bool                `json:"managementBootstrapRequired"`
	ManagementEnabled           bool                `json:"managementEnabled"`
}

func (p InstallationPlan) Values() map[string]any {
	values := map[string]any{
		"management": map[string]any{
			"enabled":       p.ManagementEnabled,
			"bootstrapOnly": p.ManagementBootstrapRequired,
		},
		"global": map[string]any{
			"corePlatformVersion": p.Release.Version,
			"clusterId":           p.ClusterID,
		},
		"runtimeAgent": map[string]any{
			"api":   map[string]any{"url": p.API.BaseURL},
			"image": map[string]any{"tag": p.Release.Version},
		},
	}
	if p.Enrollment != nil {
		values["runtimeAgent"].(map[string]any)["agent"] = map[string]any{"enrollmentToken": p.Enrollment.Token}
	}
	if p.ManagementEnabled {
		control := map[string]any{
			"api":   map[string]any{"url": p.API.BaseURL},
			"image": map[string]any{"tag": p.Release.Version},
		}
		if p.ControlEnrollment != nil {
			control["agent"] = map[string]any{"enrollmentToken": p.ControlEnrollment.Token}
		}
		values["controlAgent"] = control
	}
	return values
}

type EnrollmentAttemptStatus struct {
	Status     string `json:"status"`
	ConsumedAt string `json:"consumedAt"`
}

type AgentReport struct {
	Fresh      bool
	ReportedAt string
}

type APIError struct {
	StatusCode int
	Code       string `json:"error"`
	Message    string `json:"error_description"`
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Code != "" {
		return e.Code
	}
	return "Portal returned HTTP " + strconv.Itoa(e.StatusCode)
}

type Client struct {
	Origin     string
	HTTPClient *http.Client
	Token      string
}

func New(origin string, httpClient *http.Client, token string) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{Origin: strings.TrimRight(origin, "/"), HTTPClient: httpClient, Token: token}
}

func (c *Client) Discovery(ctx context.Context) (Discovery, error) {
	var result Discovery
	err := c.do(ctx, http.MethodGet, DiscoveryPath, nil, &result)
	if err == nil {
		if result.DeviceAuthorizationEndpoint == "" {
			result.DeviceAuthorizationEndpoint = DeviceAuthorizationPath
		}
		if result.TokenEndpoint == "" {
			result.TokenEndpoint = DeviceTokenPath
		}
		if result.UserinfoEndpoint == "" {
			result.UserinfoEndpoint = UserPath
		}
	}
	return result, err
}

func (c *Client) AuthorizeDevice(ctx context.Context, endpoint string, discovery Discovery) (DeviceAuthorization, error) {
	var result DeviceAuthorization
	err := c.do(ctx, http.MethodPost, endpoint, map[string]string{"client_id": discovery.ClientID, "scope": discovery.Scopes}, &result)
	return result, err
}

func (c *Client) PollDeviceToken(ctx context.Context, endpoint string, discovery Discovery, deviceCode string) (DeviceToken, error) {
	var result DeviceToken
	err := c.do(ctx, http.MethodPost, endpoint, map[string]string{
		"grant_type":  "urn:ietf:params:oauth:grant-type:device_code",
		"device_code": deviceCode,
		"client_id":   discovery.ClientID,
	}, &result)
	return result, err
}

func (c *Client) User(ctx context.Context, endpoint string) (User, error) {
	var result User
	err := c.do(ctx, http.MethodGet, endpoint, nil, &result)
	return result, err
}

func (c *Client) Logout(ctx context.Context) error {
	return c.do(ctx, http.MethodPost, LogoutPath, struct{}{}, nil)
}

func (c *Client) Clusters(ctx context.Context) ([]Cluster, error) {
	var result []Cluster
	err := c.do(ctx, http.MethodGet, ClustersPath, nil, &result)
	return result, err
}

func (c *Client) Cluster(ctx context.Context, id string) (Cluster, error) {
	var result Cluster
	err := c.do(ctx, http.MethodGet, ClustersPath+"/"+url.PathEscape(id), nil, &result)
	return result, err
}

func (c *Client) InstallationPlan(ctx context.Context, id string) (InstallationPlan, error) {
	return c.ClusterPlan(ctx, id, "install", "")
}

func (c *Client) ClusterPlan(ctx context.Context, id, operation, operationID string) (InstallationPlan, error) {
	var result InstallationPlan
	endpoint := map[string]string{
		"install": "installation-plan",
		"convert": "conversion-plan",
		"upgrade": "upgrade-plan",
		"repair":  "repair-plan",
	}[operation]
	if endpoint == "" {
		return result, fmt.Errorf("unsupported cluster operation %q", operation)
	}
	path := "/api/admin/connected-clusters/" + url.PathEscape(id) + "/" + endpoint
	if operationID != "" && operation != "upgrade" {
		path += "?" + url.Values{"operationId": {operationID}}.Encode()
	}
	err := c.do(ctx, http.MethodPost, path, struct{}{}, &result)
	return result, err
}

func (c *Client) EnrollmentAttemptStatus(ctx context.Context, clusterID, role, attemptID string) (EnrollmentAttemptStatus, error) {
	var result EnrollmentAttemptStatus
	path := "/api/admin/connected-clusters/" + url.PathEscape(clusterID) + "/agents/" + url.PathEscape(role) +
		"/enrollment-attempts/" + url.PathEscape(attemptID)
	err := c.do(ctx, http.MethodGet, path, nil, &result)
	return result, err
}

func (c *Client) AgentReport(ctx context.Context, clusterID, role string) (AgentReport, error) {
	if role != "runtime-agent" && role != "control-agent" {
		return AgentReport{}, fmt.Errorf("unsupported agent role %q", role)
	}
	var result struct {
		ClusterRuntime *agentRuntime `json:"clusterRuntime"`
		ClusterControl *agentRuntime `json:"clusterControl"`
	}
	path := "/api/infrastructure/environment/" + url.PathEscape(clusterID) + "?runtimeDetail=summary"
	if err := c.do(ctx, http.MethodGet, path, nil, &result); err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
			return AgentReport{}, nil
		}
		return AgentReport{}, err
	}
	runtime := result.ClusterRuntime
	if role == "control-agent" {
		runtime = result.ClusterControl
	}
	if runtime == nil {
		return AgentReport{}, nil
	}
	return AgentReport{Fresh: runtime.Freshness.Heartbeat.Status == "fresh", ReportedAt: runtime.Freshness.Heartbeat.ReportedAt}, nil
}

type agentRuntime struct {
	Freshness struct {
		Heartbeat struct {
			Status     string `json:"status"`
			ReportedAt string `json:"reportedAt"`
		} `json:"heartbeat"`
	} `json:"freshness"`
}

func (c *Client) do(ctx context.Context, method, endpoint string, body, result any) error {
	target, err := resolve(c.Origin, endpoint)
	if err != nil {
		return err
	}
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, target, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		apiErr := &APIError{StatusCode: resp.StatusCode}
		_ = json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(apiErr)
		return apiErr
	}
	if result == nil || resp.StatusCode == http.StatusNoContent {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("decode Portal response: %w", err)
	}
	return nil
}

func resolve(origin, endpoint string) (string, error) {
	base, err := url.Parse(origin)
	if err != nil {
		return "", err
	}
	relative, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}
	target := base.ResolveReference(relative)
	if target.Scheme != base.Scheme || target.Host != base.Host {
		return "", errors.New("portal discovery endpoint must use the selected instance origin")
	}
	return target.String(), nil
}
