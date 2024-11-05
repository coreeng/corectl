package env

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	. "github.com/coreeng/corectl/pkg/command"
	"github.com/coreeng/corectl/pkg/shell"

	"github.com/cedws/iapc/iap"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	"github.com/coreeng/corectl/pkg/env/proxy"
	"github.com/coreeng/corectl/pkg/gcp"
	"github.com/coreeng/developer-platform/pkg/environment"
	"github.com/phuslu/log"
	"golang.org/x/oauth2/google"
)

const BastionSquidProxyPort = 3128
const DefaultInterfaceName = "nic0"
const DefaultZone = "europe-west2-a"
const KubeNamespace = "default"

var defaultTokenScopes = []string{"https://www.googleapis.com/auth/cloud-platform"}

type EnvConnectOpts struct {
	Port               int
	Environment        *environment.Environment
	RepositoryLocation string
	ProjectID          string
	Region             string
	Streams            userio.IOStreams
	SilentExec         Commander
	Exec               Commander
	GcpClient          *gcp.Client
	Command            []string
	SkipTunnel         bool
}

// Connect establishes a connection with a gke cluster via a bastion host
func Connect(opts EnvConnectOpts) error {
	s := opts.Streams
	wizard := s.Wizard(
		"Checking platform is supported",
		"Platform is supported",
	)
	defer wizard.Done()
	if err := checkPlatformSupported(opts.Environment); err != nil {
		return err
	}

	e := opts.Environment.Platform.(*environment.GCPVendor)
	proxyUrl, err := setupConnection(s, opts.SilentExec, opts.Environment, opts.Port)
	if err != nil {
		return err
	}

	var execute func() error = nil
	log.Debug().Msgf("Commands: %+v", opts.Command)
	if len(opts.Command) > 0 {
		commandString := strings.Join(opts.Command, " ")
		log.Debug().Msgf("iap tunnel command set to: %s", commandString)
		execute = func() error {
			wizard.Info(fmt.Sprintf("Executing: %s", commandString))
			stdout, stderr, err := shell.RunCommand(".", opts.Command[0], opts.Command[1:]...)
			log.Debug().Str("command", commandString).Msgf("stdout: %s, stderr: %s", stdout, stderr)
			wizard.Print(stdout)
			if strings.Trim(string(stderr), " \t") != "" {
				s.CurrentHandler.Warn(fmt.Sprintf("stderr: %s", stderr))
			}
			return err
		}
	}
	if !opts.SkipTunnel { // solely for testing the rest of Connect - IAPC's target websocket endpoint cannot be configured
		bastionName := fmt.Sprintf("%s-bastion", opts.Environment.Environment)
		startIAPTunnel(
			s,
			e.ProjectId,
			DefaultZone,
			proxyUrl,
			bastionName,
			BastionSquidProxyPort,
			DefaultInterfaceName,
			true,
			execute,
		)
	}
	return nil
}

func startIAPTunnel(
	streams userio.IOStreams,
	project string,
	zone string,
	bind string,
	instanceName string,
	port uint16,
	interfaceName string,
	compress bool,
	execute func() error,
) {
	ctx := context.Background()

	tokenSource, err := google.DefaultTokenSource(ctx, defaultTokenScopes...)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to get default token source")
	}

	stringPort := strconv.FormatUint(uint64(port), 10)
	opts := []iap.DialOption{
		iap.WithProject(project),
		iap.WithInstance(instanceName, zone, interfaceName),
		iap.WithPort(stringPort),
		iap.WithTokenSource(&tokenSource),
	}
	log.Debug().
		Str("project", project).
		Str("instanceName", instanceName).
		Str("zone", zone).
		Str("interfaceName", interfaceName).
		Str("port", stringPort).
		Str("tokenScopes", strings.Join(defaultTokenScopes, ", ")).
		Msgf("setting iap options")
	if compress {
		opts = append(opts, iap.WithCompression())
	}

	log.Debug().Msgf("binding to %s", bind)
	proxy.Listen(streams, ctx, bind, opts, execute)
}

func setupConnection(streams userio.IOStreams, c Commander, env *environment.Environment, port int) (string, error) {
	e := env.Platform.(*environment.GCPVendor)
	wizard := streams.CurrentHandler

	wizard.SetTask(
		fmt.Sprintf("Retrieving cluster credentials: project=%s zone=%s cluster=%s", e.ProjectId, e.Region, env.Environment),
		fmt.Sprintf("Configured cluster credentials: project=%s zone=%s cluster=%s", e.ProjectId, e.Region, env.Environment),
	)
	if err := setCredentials(c, env.Environment, e.ProjectId, e.Region); err != nil {
		wizard.Abort(err.Error())
		return "", err
	}

	context := fmt.Sprintf("gke_%s_%s_%s", e.ProjectId, e.Region, env.Environment)
	wizard.SetTask(
		fmt.Sprintf("Setting Kubernetes config context to: %s", context),
		fmt.Sprintf("Kubernetes config context set to: %s", context),
	)
	if err := setKubeContext(c, context); err != nil {
		wizard.Abort(err.Error())
		return "", err
	}

	proxyUrl := fmt.Sprintf("localhost:%d", port)
	wizard.SetTask(
		fmt.Sprintf("Setting Kubernetes proxy url to: %s", proxyUrl),
		fmt.Sprintf("Kubernetes proxy url set to: %s", proxyUrl),
	)
	if err := setKubeProxy(c, context, proxyUrl); err != nil {
		wizard.Abort(err.Error())
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
	namespace := fmt.Sprintf("--namespace=%s", KubeNamespace)
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
