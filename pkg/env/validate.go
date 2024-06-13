package env

import (
	"context"
	"errors"
	"fmt"
	"os/exec"

	"github.com/coreeng/corectl/pkg/gcp"
	"github.com/coreeng/developer-platform/pkg/environment"
)

var (
	ErrCloudPlatformNotSupported = errors.New("cloud platform is not supported")
	ErrInvalidEnvironment        = errors.New("environment is not valid")
)

// Validate checks if the required tools and configurations for the environment are installed and set up correctly.
func Validate(ctx context.Context, env *environment.Environment, cmd Commander, client *gcp.Client) error {
	if env == nil {
		return ErrInvalidEnvironment
	}

	if err := depsInstalled(cmd); err != nil {
		return err
	}

	if err := platform(env); err != nil {
		return err
	}

	if err := cluster(ctx, client, env); err != nil {
		return err
	}

	return nil
}

func platform(env *environment.Environment) error {
	if env.Platform.Type() != environment.GCPVendorType {
		return ErrCloudPlatformNotSupported
	}
	return nil
}

type Commander interface {
	CommandOutput(string, ...string) ([]byte, error)
}

type Command struct {
}

func (c *Command) CommandOutput(cmd string, args ...string) ([]byte, error) {
	out, err := exec.Command(cmd, args...).Output()
	if err != nil {
		return nil, err
	}
	return out, nil
}

func depsInstalled(c Commander) error {
	var (
		kube   = "kubectl"
		gcloud = "gcloud"
	)
	if _, err := c.CommandOutput(kube); err != nil {
		return fmt.Errorf("%s is not installed: %w", kube, err)
	}
	if _, err := c.CommandOutput(gcloud); err != nil {
		return fmt.Errorf("%s is not installed: %w", gcloud, err)
	}
	return nil
}

// cluster checks if the cluster for the given environment is present in gcp.
func cluster(ctx context.Context, c *gcp.Client, env *environment.Environment) error {
	if _, err := c.GetCluster(ctx, env.Environment, env.Platform.(*environment.GCPVendor).Region, env.Platform.(*environment.GCPVendor).ProjectId); err != nil {
		return err
	}

	return nil
}
