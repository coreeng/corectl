package cluster

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/coreeng/corectl/pkg/instance"
	"github.com/coreeng/corectl/pkg/portal"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func TestConnectVerifiesFingerprintBeforeWritingManagedContext(t *testing.T) {
	connector, kubeconfigPath := testConnector(t, "actual")
	before, err := os.ReadFile(kubeconfigPath)
	require.NoError(t, err)
	remote := portal.Cluster{ID: "cluster-1", Generation: "1", KubeSystemNamespaceUID: "expected"}

	_, err = connector.Connect(context.Background(), instance.Instance{Name: "local", Origin: "https://portal.example.com"}, remote, "source", true)
	require.ErrorContains(t, err, "expected Portal fingerprint")
	after, err := os.ReadFile(kubeconfigPath)
	require.NoError(t, err)
	require.Equal(t, before, after)
}

func TestConnectCopiesReachableConfigAndReusesGenerationBinding(t *testing.T) {
	connector, kubeconfigPath := testConnector(t, "fingerprint")
	selected := instance.Instance{Name: "local", Origin: "https://portal.example.com"}
	remote := portal.Cluster{ID: "cluster-1", Generation: "1", KubeSystemNamespaceUID: "fingerprint"}

	managed, err := connector.Connect(context.Background(), selected, remote, "source", true)
	require.NoError(t, err)
	config, err := clientcmd.LoadFromFile(kubeconfigPath)
	require.NoError(t, err)
	require.Equal(t, managed, config.CurrentContext)
	require.Equal(t, "https://cluster.example.com", config.Clusters[config.Contexts[managed].Cluster].Server)
	require.Equal(t, "token", config.AuthInfos[config.Contexts[managed].AuthInfo].Token)

	reused, err := connector.Connect(context.Background(), selected, remote, "", true)
	require.NoError(t, err)
	require.Equal(t, managed, reused)

	remote.Generation = "2"
	_, err = connector.Connect(context.Background(), selected, remote, "", true)
	require.ErrorContains(t, err, "generation changed")
}

func TestConnectRefusesToOverwriteUnownedManagedNames(t *testing.T) {
	connector, kubeconfigPath := testConnector(t, "fingerprint")
	config, err := clientcmd.LoadFromFile(kubeconfigPath)
	require.NoError(t, err)
	config.Contexts["corectl-local-cluster-1"] = &clientcmdapi.Context{Cluster: "source-cluster", AuthInfo: "source-user"}
	require.NoError(t, clientcmd.WriteToFile(*config, kubeconfigPath))
	remote := portal.Cluster{ID: "cluster-1", Generation: "1", KubeSystemNamespaceUID: "fingerprint"}

	_, err = connector.Connect(context.Background(), instance.Instance{Name: "local", Origin: "https://portal.example.com"}, remote, "source", true)
	require.ErrorContains(t, err, "refusing to overwrite")
}

func TestConnectPreservesFileBackedCredentialsFromRelativePaths(t *testing.T) {
	dir := t.TempDir()
	kubeconfigPath := filepath.Join(dir, "config")
	for name, content := range map[string]string{
		"ca.pem":     "ca",
		"client.pem": "certificate",
		"client.key": "key",
		"token":      "token",
	} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600))
	}
	config := clientcmdapi.NewConfig()
	config.Clusters["source-cluster"] = &clientcmdapi.Cluster{
		Server:               "https://cluster.example.com",
		CertificateAuthority: "ca.pem",
	}
	config.AuthInfos["source-user"] = &clientcmdapi.AuthInfo{
		ClientCertificate: "client.pem",
		ClientKey:         "client.key",
		TokenFile:         "token",
	}
	config.Contexts["source"] = &clientcmdapi.Context{Cluster: "source-cluster", AuthInfo: "source-user"}
	require.NoError(t, clientcmd.WriteToFile(*config, kubeconfigPath))
	paths := clientcmd.NewDefaultPathOptions()
	paths.GlobalFile = kubeconfigPath
	paths.LoadingRules = &clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath}
	paths.LoadingRules.DoNotResolvePaths = true
	store := &instance.Store{Path: filepath.Join(dir, "platform.json")}
	client := fake.NewSimpleClientset(&corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "kube-system", UID: types.UID("fingerprint")},
	})
	connector := &Connector{
		Store:       store,
		PathOptions: paths,
		NewClient: func(clientcmd.ClientConfig) (kubernetes.Interface, error) {
			return client, nil
		},
	}

	managed, err := connector.Connect(
		context.Background(),
		instance.Instance{Name: "local", Origin: "https://portal.example.com"},
		portal.Cluster{ID: "cluster-1", Generation: "1", KubeSystemNamespaceUID: "fingerprint"},
		"source",
		false,
	)
	require.NoError(t, err)
	written, err := clientcmd.LoadFromFile(kubeconfigPath)
	require.NoError(t, err)
	managedContext := written.Contexts[managed]
	managedCluster := written.Clusters[managedContext.Cluster]
	managedAuth := written.AuthInfos[managedContext.AuthInfo]
	require.Equal(t, filepath.Join(dir, "ca.pem"), managedCluster.CertificateAuthority)
	require.Equal(t, filepath.Join(dir, "client.pem"), managedAuth.ClientCertificate)
	require.Equal(t, filepath.Join(dir, "client.key"), managedAuth.ClientKey)
	require.Equal(t, filepath.Join(dir, "token"), managedAuth.TokenFile)
}

func testConnector(t *testing.T, namespaceUID string) (*Connector, string) {
	t.Helper()
	dir := t.TempDir()
	kubeconfigPath := filepath.Join(dir, "config")
	config := clientcmdapi.NewConfig()
	config.Clusters["source-cluster"] = &clientcmdapi.Cluster{Server: "https://cluster.example.com"}
	config.AuthInfos["source-user"] = &clientcmdapi.AuthInfo{Token: "token"}
	config.Contexts["source"] = &clientcmdapi.Context{Cluster: "source-cluster", AuthInfo: "source-user", Namespace: "default"}
	config.CurrentContext = "source"
	require.NoError(t, clientcmd.WriteToFile(*config, kubeconfigPath))
	paths := clientcmd.NewDefaultPathOptions()
	paths.GlobalFile = kubeconfigPath
	paths.LoadingRules = &clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath}
	store := &instance.Store{Path: filepath.Join(dir, "platform.json")}
	client := fake.NewSimpleClientset(&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "kube-system", UID: types.UID(namespaceUID)}})
	return &Connector{
		Store:       store,
		PathOptions: paths,
		NewClient: func(clientcmd.ClientConfig) (kubernetes.Interface, error) {
			return client, nil
		},
	}, kubeconfigPath
}
