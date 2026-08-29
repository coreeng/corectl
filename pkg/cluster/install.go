package cluster

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/coreeng/corectl/pkg/portal"
	ocidigest "github.com/opencontainers/go-digest"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/registry"
	"helm.sh/helm/v3/pkg/storage/driver"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	orasregistry "oras.land/oras-go/v2/registry"
)

const DefaultInstallTimeout = 10 * time.Minute

type Installer interface {
	Verify(ctx context.Context, expectedFingerprint, kubeContext string) error
	Apply(ctx context.Context, plan portal.InstallationPlan, kubeContext string, timeout time.Duration) error
}

type HelmInstaller struct {
	Kubeconfig string
}

func (h HelmInstaller) Verify(ctx context.Context, expectedFingerprint, kubeContext string) error {
	if kubeContext == "" {
		return fmt.Errorf("kubeconfig context is required")
	}
	flags := genericclioptions.NewConfigFlags(false)
	flags.Context = &kubeContext
	if h.Kubeconfig != "" {
		flags.KubeConfig = &h.Kubeconfig
	}
	return verifyInstallTarget(ctx, flags, expectedFingerprint)
}

func (h HelmInstaller) Apply(ctx context.Context, plan portal.InstallationPlan, kubeContext string, timeout time.Duration) error {
	if kubeContext == "" {
		return fmt.Errorf("kubeconfig context is required")
	}
	if plan.Release.Name == "" || plan.Release.Namespace == "" || plan.Release.Version == "" ||
		plan.API.BaseURL == "" || plan.Chart.Reference == "" || plan.Chart.Version == "" {
		return fmt.Errorf("portal installation plan is incomplete")
	}
	if err := validateEnrollments(plan); err != nil {
		return err
	}
	if !strings.HasPrefix(plan.Chart.Reference, "oci://") {
		return fmt.Errorf("portal chart reference %q is not OCI", plan.Chart.Reference)
	}
	ref, err := exactChartReference(plan.Chart)
	if err != nil {
		return err
	}
	flags := genericclioptions.NewConfigFlags(false)
	flags.Context = &kubeContext
	flags.Namespace = &plan.Release.Namespace
	if h.Kubeconfig != "" {
		flags.KubeConfig = &h.Kubeconfig
	}
	if err := verifyInstallTarget(ctx, flags, plan.ClusterFingerprint); err != nil {
		return err
	}
	registryClient, err := registry.NewClient(registry.ClientOptHTTPClient(&http.Client{Timeout: timeout}))
	if err != nil {
		return fmt.Errorf("create Helm registry client: %w", err)
	}
	pulled, err := registryClient.Pull(strings.TrimPrefix(ref, "oci://"), registry.PullOptWithChart(true))
	if err != nil {
		return fmt.Errorf("pull Helm chart %s: %w", ref, err)
	}
	if plan.Chart.Digest != "" && (pulled.Manifest == nil || pulled.Manifest.Digest != plan.Chart.Digest) {
		actual := ""
		if pulled.Manifest != nil {
			actual = pulled.Manifest.Digest
		}
		return fmt.Errorf("helm chart digest mismatch: got %q, expected %q", actual, plan.Chart.Digest)
	}
	if pulled.Chart == nil || pulled.Chart.Meta == nil || pulled.Chart.Meta.Version != plan.Chart.Version {
		return fmt.Errorf("helm chart metadata does not match requested version %q", plan.Chart.Version)
	}
	chart, err := loader.LoadArchive(bytes.NewReader(pulled.Chart.Data))
	if err != nil {
		return fmt.Errorf("load pulled Helm chart: %w", err)
	}
	configuration := new(action.Configuration)
	if err := configuration.Init(flags, plan.Release.Namespace, "secret", func(format string, args ...any) {}); err != nil {
		return fmt.Errorf("initialize Helm: %w", err)
	}
	history := action.NewHistory(configuration)
	history.Max = 1
	runUpgrade := func(values map[string]any) error {
		upgrade := action.NewUpgrade(configuration)
		upgrade.SetRegistryClient(registryClient)
		upgrade.Namespace = plan.Release.Namespace
		upgrade.ResetValues = true
		upgrade.Wait = true
		upgrade.WaitForJobs = true
		upgrade.Timeout = timeout
		_, err := upgrade.RunWithContext(ctx, plan.Release.Name, chart, values)
		return err
	}
	if _, err := history.Run(plan.Release.Name); err != nil {
		if !errors.Is(err, driver.ErrReleaseNotFound) {
			return fmt.Errorf("inspect Helm release %s: %w", plan.Release.Name, err)
		}
		if plan.Operation != "install" && plan.Operation != "repair" {
			return fmt.Errorf("%s requires existing Helm release %s", plan.Operation, plan.Release.Name)
		}
		install := action.NewInstall(configuration)
		install.SetRegistryClient(registryClient)
		install.ReleaseName = plan.Release.Name
		install.Namespace = plan.Release.Namespace
		install.CreateNamespace = true
		install.Wait = true
		install.WaitForJobs = true
		install.Timeout = timeout
		if _, err := install.RunWithContext(ctx, chart, plan.Values()); err != nil {
			return fmt.Errorf("install %s: %w", plan.Release.Name, err)
		}
	} else if err := runUpgrade(plan.Values()); err != nil {
		return fmt.Errorf("upgrade %s: %w", plan.Release.Name, err)
	}
	if plan.ManagementBootstrapRequired {
		plan.ManagementBootstrapRequired = false
		if err := runUpgrade(plan.Values()); err != nil {
			return fmt.Errorf("complete management bootstrap for %s: %w", plan.Release.Name, err)
		}
	}
	return nil
}

func validateEnrollments(plan portal.InstallationPlan) error {
	valid := func(enrollment *portal.EnrollmentMetadata, role string) bool {
		return enrollment != nil && enrollment.Role == role && enrollment.Token != "" && enrollment.AttemptID != ""
	}
	switch plan.Operation {
	case "install", "repair":
		if !valid(plan.Enrollment, "runtime-agent") {
			return fmt.Errorf("portal %s plan has no Runtime Agent enrollment", plan.Operation)
		}
		if plan.ManagementEnabled && !valid(plan.ControlEnrollment, "control-agent") {
			return fmt.Errorf("portal managed %s plan has no Control Agent enrollment", plan.Operation)
		}
	case "convert":
		if plan.Enrollment != nil || !plan.ManagementEnabled || !valid(plan.ControlEnrollment, "control-agent") {
			return fmt.Errorf("portal conversion plan is incomplete")
		}
	case "upgrade":
		if plan.Enrollment != nil || plan.ControlEnrollment != nil {
			return fmt.Errorf("portal upgrade plan unexpectedly contains enrollment material")
		}
	default:
		return fmt.Errorf("unsupported cluster operation %q", plan.Operation)
	}
	return nil
}

func verifyInstallTarget(ctx context.Context, flags *genericclioptions.ConfigFlags, expectedFingerprint string) error {
	restConfig, err := flags.ToRESTConfig()
	if err != nil {
		return fmt.Errorf("load Kubernetes context for fingerprint verification: %w", err)
	}
	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("create Kubernetes client for fingerprint verification: %w", err)
	}
	namespace, err := client.CoreV1().Namespaces().Get(ctx, "kube-system", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("read kube-system for fingerprint verification: %w", err)
	}
	if expectedFingerprint == "" {
		return nil
	}
	if string(namespace.UID) != expectedFingerprint {
		return fmt.Errorf("installation context points to kube-system UID %q, expected Portal fingerprint %q", namespace.UID, expectedFingerprint)
	}
	return nil
}

func exactChartReference(chart portal.Chart) (string, error) {
	parsed, err := orasregistry.ParseReference(strings.TrimPrefix(chart.Reference, "oci://"))
	if err != nil {
		return "", fmt.Errorf("invalid Portal OCI chart reference %q: %w", chart.Reference, err)
	}
	if parsed.Reference != "" && parsed.Reference != chart.Version && parsed.Reference != chart.Digest {
		return "", fmt.Errorf("portal chart reference selector %q conflicts with version %q and digest %q", parsed.Reference, chart.Version, chart.Digest)
	}
	selector := chart.Version
	separator := ":"
	if chart.Digest != "" {
		digest, err := ocidigest.Parse(chart.Digest)
		if err != nil {
			return "", fmt.Errorf("invalid Portal OCI chart digest %q: %w", chart.Digest, err)
		}
		selector = digest.String()
		separator = "@"
	}
	return "oci://" + parsed.Registry + "/" + parsed.Repository + separator + selector, nil
}
