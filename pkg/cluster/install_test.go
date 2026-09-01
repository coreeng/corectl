package cluster

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/coreeng/corectl/pkg/portal"
	"github.com/stretchr/testify/require"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func TestExactChartReferencePinsPortalDigest(t *testing.T) {
	chart := portal.Chart{Reference: "oci://ghcr.io/coreeng/agent", Version: "1.2.3", Digest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
	ref, err := exactChartReference(chart)
	require.NoError(t, err)
	require.Equal(t, "oci://ghcr.io/coreeng/agent@"+chart.Digest, ref)

	chart.Reference += ":9.9.9"
	_, err = exactChartReference(chart)
	require.ErrorContains(t, err, "conflicts")
}

func TestVerifyInstallTargetRejectsDifferentClusterFingerprint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/v1/namespaces/kube-system", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"apiVersion":"v1","kind":"Namespace","metadata":{"name":"kube-system","uid":"actual"}}`)
	}))
	defer server.Close()
	path := filepath.Join(t.TempDir(), "config")
	config := clientcmdapi.NewConfig()
	config.Clusters["cluster"] = &clientcmdapi.Cluster{Server: server.URL}
	config.AuthInfos["user"] = &clientcmdapi.AuthInfo{}
	config.Contexts["source"] = &clientcmdapi.Context{Cluster: "cluster", AuthInfo: "user"}
	require.NoError(t, clientcmd.WriteToFile(*config, path))
	flags := genericclioptions.NewConfigFlags(false)
	flags.KubeConfig = &path
	contextName := "source"
	flags.Context = &contextName

	err := verifyInstallTarget(context.Background(), flags, "expected")

	require.ErrorContains(t, err, "expected Portal fingerprint")
}

func TestVerifyInstallTargetContactsClusterWithoutPortalFingerprint(t *testing.T) {
	contacted := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contacted = true
		require.Equal(t, "/api/v1/namespaces/kube-system", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"apiVersion":"v1","kind":"Namespace","metadata":{"name":"kube-system","uid":"actual"}}`)
	}))
	defer server.Close()
	path := filepath.Join(t.TempDir(), "config")
	config := clientcmdapi.NewConfig()
	config.Clusters["cluster"] = &clientcmdapi.Cluster{Server: server.URL}
	config.AuthInfos["user"] = &clientcmdapi.AuthInfo{}
	config.Contexts["source"] = &clientcmdapi.Context{Cluster: "cluster", AuthInfo: "user"}
	require.NoError(t, clientcmd.WriteToFile(*config, path))
	flags := genericclioptions.NewConfigFlags(false)
	flags.KubeConfig = &path
	contextName := "source"
	flags.Context = &contextName

	require.NoError(t, verifyInstallTarget(context.Background(), flags, ""))
	require.True(t, contacted)
}

func TestExactChartReferenceUsesPortalVersionWithoutDigest(t *testing.T) {
	chart := portal.Chart{Reference: "oci://ghcr.io/coreeng/charts/core-platform-cluster-system", Version: "1.2.3"}
	ref, err := exactChartReference(chart)
	require.NoError(t, err)
	require.Equal(t, "oci://ghcr.io/coreeng/charts/core-platform-cluster-system:1.2.3", ref)
}
