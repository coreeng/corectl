package env

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/cedws/iapc/iap"
	"github.com/coreeng/core-platform/pkg/environment"
	"github.com/coreeng/corectl/pkg/cmdutil/userio"
	. "github.com/coreeng/corectl/pkg/command"
	"github.com/coreeng/corectl/pkg/gcp"
	"github.com/coreeng/corectl/pkg/logger"
	"github.com/coreeng/corectl/pkg/shell"
	"go.uber.org/zap"
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
	Background         bool
	Force              bool
}

// Connect establishes a connection with a gke cluster via a bastion host
func Connect(opts EnvConnectOpts) error {
	s := opts.Streams

	if opts.Port == 0 {
		opts.Port = GenerateConnectPort(opts.Environment.Environment)
	}
	if IsConnectStartup(opts) {
		existingPid := ExistingPidForConnection(opts.Environment.Environment)
		if existingPid != 0 {
			if !opts.Force {
				logger.Warn().Msgf("Proxy for %s already running with pid %d", opts.Environment.Environment, existingPid)
				return nil
			}
			if err := KillProcess(opts.Environment.Environment, int32(existingPid), false); err != nil {
				return fmt.Errorf("[%s] failed to kill process: %w", opts.Environment.Environment, err)
			}
		}

		logger.Info().Msg("Checking platform is supported")
		defer logger.Info().Msg("Platform is supported")

		if err := checkPlatformSupported(opts.Environment); err != nil {
			return err
		}
	}

	e := opts.Environment.Platform.(*environment.GCPVendor)
	// Only run startup if we are in the foreground or in the background and parent process
	proxyUrl, err := setupConnection(s, opts, opts.SilentExec, opts.Environment, opts.Port)
	if err != nil {
		return err
	}

	var execute func() error = nil
	logger.Debug().Msgf("Commands: %+v", opts.Command)
	if len(opts.Command) > 0 {
		commandString := strings.Join(opts.Command, " ")
		logger.Debug().Msgf("iap tunnel command set to: %s", commandString)
		execute = func() error {
			stdout, stderr, err := shell.RunCommand(".", opts.Command[0], opts.Command[1:]...)
			logger.Debug().With(zap.String("command", commandString)).Msgf("stdout: %s, stderr: %s", stdout, stderr)
			if strings.Trim(string(stderr), " \t") != "" {
				s.CurrentHandler.Warn(fmt.Sprintf("stderr: %s", stderr))
			}
			return err
		}
	}
	if !opts.SkipTunnel { // solely for testing the rest of Connect - IAPC's target websocket endpoint cannot be configured
		bastionName := fmt.Sprintf("%s-bastion", opts.Environment.Environment)
		startIAPTunnel(
			opts,
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
	opts EnvConnectOpts,
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
		logger.Fatal().With(zap.Error(err)).Msg("failed to get default token source")
	}

	stringPort := strconv.FormatUint(uint64(port), 10)
	dialOpts := []iap.DialOption{
		iap.WithProject(project),
		iap.WithInstance(instanceName, zone, interfaceName),
		iap.WithPort(stringPort),
		iap.WithTokenSource(&tokenSource),
	}
	logger.Debug().With(
		zap.String("project", project),
		zap.String("instanceName", instanceName),
		zap.String("zone", zone),
		zap.String("interfaceName", interfaceName),
		zap.String("port", stringPort),
		zap.String("tokenScopes", strings.Join(defaultTokenScopes, ", "))).
		Msgf("setting iap options")
	if compress {
		dialOpts = append(dialOpts, iap.WithCompression())
	}

	logger.Debug().Msgf("binding to %s", bind)
	Listen(streams, opts, ctx, bind, dialOpts, execute)
}

func setupConnection(streams userio.IOStreams, opts EnvConnectOpts, c Commander, env *environment.Environment, port int) (string, error) {
	e := env.Platform.(*environment.GCPVendor)
	// TODO: We need to make proxy URL more dynamic
	proxyUrl := fmt.Sprintf("localhost:%d", port)
	if !IsConnectStartup(opts) {
		return proxyUrl, nil
	}

	logger.Info().Msgf("Retrieving cluster credentials: project=%s zone=%s cluster=%s", e.ProjectId, e.Region, env.Environment)

	if err := setCredentials(c, env.Environment, e.ProjectId, e.Region); err != nil {
		logger.Error().Msg(err.Error())
		return "", err
	}
	logger.Info().Msgf("Configured cluster credentials: project=%s zone=%s cluster=%s", e.ProjectId, e.Region, env.Environment)

	context := fmt.Sprintf("gke_%s_%s_%s", e.ProjectId, e.Region, env.Environment)
	logger.Info().Msgf("Setting Kubernetes config context to: %s", context)

	if err := setKubeContext(c, context); err != nil {
		logger.Error().Msg(err.Error())
		return "", err
	}
	logger.Warn().Msgf("Kubernetes config context set to: %s", context)

	logger.Info().Msgf("Setting Kubernetes proxy url to: %s", proxyUrl)
	if err := setKubeProxy(c, context, proxyUrl); err != nil {
		logger.Error().Msg(err.Error())
		return "", err
	}

	logger.Info().Msgf("Kubernetes proxy url set to: %s", proxyUrl)
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
