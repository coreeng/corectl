package env

import (
	"fmt"

	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/coreeng/developer-platform/pkg/environment"
)

var kubeNamespace = "default"

// Connect establishes a connection with a gke cluster via a bastion host
func Connect(stream userio.IOStreams, env *environment.Environment, c Commander, projectID, region string, proxyPort int) error {
	cluster := env.Environment
	// generate credentials and update kubeconfig
	if err := credentials(c, cluster, projectID, region); err != nil {
		return err
	}
	// set kube context
	context := fmt.Sprintf("gke_%s_%s_%s", projectID, region, cluster)
	if err := kubeContext(c, context); err != nil {
		return err
	}
	// setup kube proxy with bastion
	proxyUrl := fmt.Sprintf("localhost:%d", proxyPort)
	if err := proxy(c, context, proxyUrl); err != nil {
		return err
	}
	// setup iap tunnel with bastion
	stream.Info("You may now use kubectl to query resources in the cluster. Keep this running in the background.")
	if err := iapTunnel(c, cluster, projectID, proxyUrl); err != nil {
		return err
	}

	return nil
}

func credentials(c Commander, cluster, projectID, region string) error {
	if _, err := c.Execute("gcloud", "container", "clusters", "get-credentials", "--project", projectID, "--zone", region, "--internal-ip", cluster); err != nil {
		return err
	}
	return nil
}

func kubeContext(c Commander, context string) error {
	namespace := fmt.Sprintf("--namespace=%s", kubeNamespace)
	if _, err := c.Execute("kubectl", "config", "set-context", context, namespace); err != nil {
		return err
	}
	return nil
}

func proxy(c Commander, context, proxy string) error {
	url := fmt.Sprintf("clusters.%s.proxy-url", context)
	if _, err := c.Execute("kubectl", "config", "set", url, "http://"+proxy); err != nil {
		return err
	}
	return nil
}

func iapTunnel(c Commander, env, projectID, proxy string) error {
	if _, err := c.Execute("gcloud", "compute", "start-iap-tunnel", env+"-bastion", "3128", "--local-host-port", proxy, "--project", projectID, "--zone", "europe-west2-a"); err != nil {
		return err
	}
	return nil
}
