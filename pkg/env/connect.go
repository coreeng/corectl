package env

import (
	"fmt"

	. "github.com/coreeng/corectl/pkg/command"

	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/coreeng/developer-platform/pkg/environment"
)

var kubeNamespace = "default"

// Connect establishes a connection with a gke cluster via a bastion host
func Connect(s userio.IOStreams, env *environment.Environment, c Commander, port int) error {
	if err := checkPlatformSupported(env); err != nil {
		return err
	}
	e := env.Platform.(*environment.GCPVendor)
	proxyUrl, err := setupConnection(s, c, env, port)
	if err != nil {
		return err
	}
	// setup iap tunnel with bastion
	s.Info("You may now use kubectl to query resources in the cluster. Keep this running in the background.")
	if err := startIAPTunnel(c, env.Environment, e.ProjectId, proxyUrl); err != nil {
		return err
	}

	return nil
}

func setupConnection(streams userio.IOStreams, c Commander, env *environment.Environment, port int) (string, error) {
	streams.Wizard(
		fmt.Sprintf("Connecting to cluster: %s (%s)", env.Environment, env.Domain),
		fmt.Sprintf("Connected to cluster: %s (%s)", env.Environment, env.Domain),
	)
	// TODO: update to use wizard for Info calls and prompts
	defer streams.CurrentHandler.Done()

	e := env.Platform.(*environment.GCPVendor)
	// generate credentials and update kubeconfig
	if err := setCredentials(c, env.Environment, e.ProjectId, e.Region); err != nil {
		return "", err
	}
	// set kube context
	context := fmt.Sprintf("gke_%s_%s_%s", e.ProjectId, e.Region, env.Environment)
	if err := setKubeContext(c, context); err != nil {
		return "", err
	}
	// setup kube proxy with bastion
	proxyUrl := fmt.Sprintf("localhost:%d", port)
	if err := setKubeProxy(c, context, proxyUrl); err != nil {
		return "", err
	}
	return proxyUrl, nil
}

func setCredentials(c Commander, cluster, projectID, region string) error {
	if _, err := c.Execute("gcloud", WithArgs("container", "clusters", "get-credentials", "--project", projectID, "--zone", region, "--internal-ip", cluster)); err != nil {
		return fmt.Errorf("get gcp cluster credentials: %w", err)
	}
	return nil
}

func setKubeContext(c Commander, context string) error {
	namespace := fmt.Sprintf("--namespace=%s", kubeNamespace)
	if _, err := c.Execute("kubectl", WithArgs("config", "set-context", context, namespace)); err != nil {
		return fmt.Errorf("set kube context %q: %w", context, err)
	}
	return nil
}

func setKubeProxy(c Commander, context, proxy string) error {
	url := fmt.Sprintf("clusters.%s.proxy-url", context)
	if _, err := c.Execute("kubectl", WithArgs("config", "set", url, "http://"+proxy)); err != nil {
		return fmt.Errorf("set kube proxy %q: %w", proxy, err)
	}
	return nil
}

func startIAPTunnel(c Commander, env, projectID, proxy string) error {
	if _, err := c.Execute("gcloud", WithArgs("compute", "start-iap-tunnel", env+"-bastion", "3128", "--local-host-port", proxy, "--project", projectID, "--zone", "europe-west2-a")); err != nil {
		return fmt.Errorf("establishing connection to IAP tunnel: %w", err)
	}
	return nil
}
