package portal

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClientUsesCentralEndpointsAndBearerToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer secret", r.Header.Get("Authorization"))
		switch r.URL.Path {
		case DiscoveryPath:
			_ = json.NewEncoder(w).Encode(Discovery{ClientID: "corectl"})
		case ClustersPath:
			_ = json.NewEncoder(w).Encode([]Cluster{{ID: "one"}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := New(server.URL, server.Client(), "secret")
	discovery, err := client.Discovery(context.Background())
	require.NoError(t, err)
	require.Equal(t, DeviceAuthorizationPath, discovery.DeviceAuthorizationEndpoint)
	clusters, err := client.Clusters(context.Background())
	require.NoError(t, err)
	require.Equal(t, "one", clusters[0].ID)
}

func TestDiscoveryCannotRedirectCredentialsToAnotherOrigin(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(Discovery{UserinfoEndpoint: "https://attacker.example/user"})
	}))
	defer server.Close()

	client := New(server.URL, server.Client(), "secret")
	discovery, err := client.Discovery(context.Background())
	require.NoError(t, err)
	_, err = client.User(context.Background(), discovery.UserinfoEndpoint)
	require.ErrorContains(t, err, "selected instance origin")
}

func TestGenerationAcceptsPortalStringOrNumber(t *testing.T) {
	for input, expected := range map[string]Generation{`"revision-1"`: "revision-1", `42`: "42"} {
		var generation Generation
		require.NoError(t, json.Unmarshal([]byte(input), &generation))
		require.Equal(t, expected, generation)
	}
}

func TestPortalInstallationPlanContractBuildsClusterSystemValues(t *testing.T) {
	var plan InstallationPlan
	require.NoError(t, json.Unmarshal([]byte(`{
		"clusterId":"local-dev",
		"generation":"generation-1",
		"clusterFingerprint":"namespace-uid",
		"api":{"baseUrl":"https://portal.example/api"},
		"release":{"name":"core-platform-cluster-system","namespace":"core-platform-system","version":"v1.2.3"},
		"chart":{"reference":"oci://ghcr.io/coreeng/charts/core-platform-cluster-system","version":"1.2.3"},
		"enrollment":{"role":"runtime-agent","token":"one-use","expiresAt":"2026-08-28T10:15:00Z","attemptId":"attempt"},
		"managementEnabled":false
	}`), &plan))
	require.Equal(t, Generation("generation-1"), plan.Generation)
	require.Equal(t, "core-platform-system", plan.Release.Namespace)
	values := plan.Values()
	require.Equal(t, "local-dev", values["global"].(map[string]any)["clusterId"])
	require.Equal(t, "one-use", values["runtimeAgent"].(map[string]any)["agent"].(map[string]any)["enrollmentToken"])
}

func TestManagedInstallationPlanBuildsControlAgentValues(t *testing.T) {
	plan := InstallationPlan{
		ClusterID:                   "managed",
		API:                         APIMetadata{BaseURL: "https://portal.example/api"},
		Release:                     ReleaseMetadata{Version: "v1.2.3"},
		Enrollment:                  &EnrollmentMetadata{Token: "runtime-token"},
		ControlEnrollment:           &EnrollmentMetadata{Token: "control-token"},
		ManagementBootstrapRequired: true,
		ManagementEnabled:           true,
	}

	values := plan.Values()
	control := values["controlAgent"].(map[string]any)
	require.Equal(t, true, values["management"].(map[string]any)["bootstrapOnly"])
	require.Equal(t, "https://portal.example/api", control["api"].(map[string]any)["url"])
	require.Equal(t, "control-token", control["agent"].(map[string]any)["enrollmentToken"])
}

func TestClusterPlanRoutesOperationsAndParsesNullableEnrollments(t *testing.T) {
	wantedPaths := []string{
		"/api/admin/connected-clusters/one/installation-plan",
		"/api/admin/connected-clusters/one/conversion-plan",
		"/api/admin/connected-clusters/one/upgrade-plan",
		"/api/admin/connected-clusters/one/repair-plan",
	}
	operationByPath := map[string]string{
		wantedPaths[0]: "install",
		wantedPaths[1]: "convert",
		wantedPaths[2]: "upgrade",
		wantedPaths[3]: "repair",
	}
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		paths = append(paths, r.URL.Path)
		_ = json.NewEncoder(w).Encode(InstallationPlan{Operation: operationByPath[r.URL.Path], ClusterID: "one"})
	}))
	defer server.Close()

	client := New(server.URL, server.Client(), "secret")
	for _, operation := range []string{"install", "convert", "upgrade", "repair"} {
		plan, err := client.ClusterPlan(context.Background(), "one", operation)
		require.NoError(t, err)
		require.Equal(t, operation, plan.Operation)
		require.Nil(t, plan.Enrollment)
		require.Nil(t, plan.ControlEnrollment)
	}
	require.Equal(t, wantedPaths, paths)

	_, err := client.ClusterPlan(context.Background(), "one", "unknown")
	require.ErrorContains(t, err, "unsupported cluster operation")
}

func TestAgentReportReadsRuntimeAndControlHeartbeat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/infrastructure/environment/one", r.URL.Path)
		require.Equal(t, "summary", r.URL.Query().Get("runtimeDetail"))
		_, _ = w.Write([]byte(`{
			"clusterRuntime":{"freshness":{"heartbeat":{"status":"fresh","reportedAt":"2026-08-29T10:00:00Z"}}},
			"clusterControl":{"freshness":{"heartbeat":{"status":"stale","reportedAt":"2026-08-29T09:00:00Z"}}}
		}`))
	}))
	defer server.Close()

	client := New(server.URL, server.Client(), "secret")
	runtime, err := client.AgentReport(context.Background(), "one", "runtime-agent")
	require.NoError(t, err)
	require.Equal(t, AgentReport{Fresh: true, ReportedAt: "2026-08-29T10:00:00Z"}, runtime)
	control, err := client.AgentReport(context.Background(), "one", "control-agent")
	require.NoError(t, err)
	require.Equal(t, AgentReport{Fresh: false, ReportedAt: "2026-08-29T09:00:00Z"}, control)
}
