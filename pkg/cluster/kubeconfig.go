package cluster

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/coreeng/corectl/pkg/instance"
	"github.com/coreeng/corectl/pkg/portal"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

type Connector struct {
	Store       *instance.Store
	PathOptions *clientcmd.PathOptions
	NewClient   func(clientcmd.ClientConfig) (kubernetes.Interface, error)
}

func NewConnector(store *instance.Store) *Connector {
	return &Connector{
		Store:       store,
		PathOptions: clientcmd.NewDefaultPathOptions(),
		NewClient: func(config clientcmd.ClientConfig) (kubernetes.Interface, error) {
			restConfig, err := config.ClientConfig()
			if err != nil {
				return nil, err
			}
			return kubernetes.NewForConfig(restConfig)
		},
	}
}

func (c *Connector) Connect(ctx context.Context, selected instance.Instance, remote portal.Cluster, sourceContext string, switchContext bool) (string, error) {
	if remote.KubeSystemNamespaceUID == "" || remote.Generation == "" {
		return "", fmt.Errorf("Portal cluster %q is missing its fingerprint or generation", remote.ID)
	}
	binding, bound, err := c.Store.Binding(selected.Origin, remote.ID)
	if err != nil {
		return "", err
	}
	config, err := c.PathOptions.GetStartingConfig()
	if err != nil {
		return "", fmt.Errorf("load kubeconfig: %w", err)
	}
	if bound && binding.Generation == string(remote.Generation) && sourceContext == "" {
		if _, ok := config.Contexts[binding.ManagedContext]; !ok {
			return "", fmt.Errorf("managed context %q is missing; reconnect with --context", binding.ManagedContext)
		}
		if err := c.verifyFingerprint(ctx, *config, binding.ManagedContext, remote.KubeSystemNamespaceUID); err != nil {
			return "", err
		}
		if switchContext {
			config.CurrentContext = binding.ManagedContext
			if err := clientcmd.ModifyConfig(c.PathOptions, *config, false); err != nil {
				return "", fmt.Errorf("switch kubeconfig context: %w", err)
			}
		}
		return binding.ManagedContext, nil
	}
	if sourceContext == "" {
		return "", errorsForContext(bound)
	}
	source, ok := config.Contexts[sourceContext]
	if !ok {
		return "", fmt.Errorf("kubeconfig context %q does not exist", sourceContext)
	}
	if err := c.verifyFingerprint(ctx, *config, sourceContext, remote.KubeSystemNamespaceUID); err != nil {
		return "", err
	}
	clusterName := managedName(selected.Name, remote.ID, "cluster")
	authName := managedName(selected.Name, remote.ID, "user")
	contextName := managedName(selected.Name, remote.ID, "")
	owned := bound && binding.ManagedContext == contextName
	if !owned && (config.Clusters[clusterName] != nil || config.AuthInfos[authName] != nil || config.Contexts[contextName] != nil) {
		return "", fmt.Errorf("refusing to overwrite existing kubeconfig entries for managed context %q", contextName)
	}
	if owned {
		if existing := config.Contexts[contextName]; existing != nil && (existing.Cluster != clusterName || existing.AuthInfo != authName) {
			return "", fmt.Errorf("managed context %q was modified; refusing to overwrite it", contextName)
		}
	}
	sourceCluster, ok := config.Clusters[source.Cluster]
	if !ok {
		return "", fmt.Errorf("context %q references missing cluster %q", sourceContext, source.Cluster)
	}
	sourceAuth, ok := config.AuthInfos[source.AuthInfo]
	if !ok {
		return "", fmt.Errorf("context %q references missing auth info %q", sourceContext, source.AuthInfo)
	}
	managedCluster := sourceCluster.DeepCopy()
	managedAuth := sourceAuth.DeepCopy()
	resolved := clientcmdapi.NewConfig()
	resolved.Clusters[source.Cluster] = managedCluster
	resolved.AuthInfos[source.AuthInfo] = managedAuth
	if err := clientcmd.ResolveLocalPaths(resolved); err != nil {
		return "", fmt.Errorf("resolve kubeconfig file references: %w", err)
	}
	managedCluster.LocationOfOrigin = ""
	config.Clusters[clusterName] = managedCluster
	managedAuth.LocationOfOrigin = ""
	config.AuthInfos[authName] = managedAuth
	managedContext := source.DeepCopy()
	managedContext.LocationOfOrigin = ""
	managedContext.Cluster = clusterName
	managedContext.AuthInfo = authName
	config.Contexts[contextName] = managedContext
	if err := c.Store.SetBinding(selected.Origin, remote.ID, instance.Binding{Generation: string(remote.Generation), ManagedContext: contextName}); err != nil {
		return "", err
	}
	// Claim the generated names before writing so a partial kubeconfig write can
	// be safely retried with --context rather than appearing to be unowned.
	if err := clientcmd.ModifyConfig(c.PathOptions, *config, false); err != nil {
		return "", fmt.Errorf("write managed kubeconfig context: %w", err)
	}
	if switchContext {
		config.CurrentContext = contextName
		if err := clientcmd.ModifyConfig(c.PathOptions, *config, false); err != nil {
			return "", fmt.Errorf("switch kubeconfig context: %w", err)
		}
	}
	return contextName, nil
}

func (c *Connector) verifyFingerprint(ctx context.Context, config clientcmdapi.Config, contextName, expected string) error {
	resolved := config.DeepCopy()
	if err := clientcmd.ResolveLocalPaths(resolved); err != nil {
		return fmt.Errorf("resolve kubeconfig file references: %w", err)
	}
	clientConfig := clientcmd.NewNonInteractiveClientConfig(*resolved, contextName, &clientcmd.ConfigOverrides{}, c.PathOptions)
	client, err := c.NewClient(clientConfig)
	if err != nil {
		return fmt.Errorf("create Kubernetes client for context %q: %w", contextName, err)
	}
	namespace, err := client.CoreV1().Namespaces().Get(ctx, "kube-system", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("read kube-system through context %q: %w", contextName, err)
	}
	if string(namespace.UID) != expected {
		return fmt.Errorf("context %q points to kube-system UID %q, expected Portal fingerprint %q", contextName, namespace.UID, expected)
	}
	return nil
}

func errorsForContext(generationChanged bool) error {
	if generationChanged {
		return fmt.Errorf("cluster generation changed; provide the verified existing kubeconfig --context")
	}
	return fmt.Errorf("first connection requires an explicit existing kubeconfig --context")
}

var unsafeName = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func managedName(instanceName, clusterID, suffix string) string {
	name := "corectl-" + unsafeName.ReplaceAllString(instanceName, "-") + "-" + unsafeName.ReplaceAllString(clusterID, "-")
	if suffix != "" {
		name += "-" + suffix
	}
	return strings.Trim(name, "-")
}
