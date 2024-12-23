package env

import (
	"context"
	"errors"
	"fmt"
	"github.com/coreeng/corectl/pkg/command"
	"strings"

	"github.com/coreeng/core-platform/pkg/environment"
	"github.com/coreeng/corectl/pkg/gcp"
)

var (
	ErrCloudPlatformNotSupported = errors.New("cloud platform is not supported")
	ErrInvalidEnvironment        = errors.New("environment is not valid")
)

// Validate checks if the required tools and configurations for the environment are installed and set up correctly.
func Validate(ctx context.Context, env *environment.Environment, cmd command.Commander, client *gcp.Client) error {
	if env == nil {
		return ErrInvalidEnvironment
	}

	if err := command.DepsInstalled(cmd, "gcloud", "kubectl"); err != nil {
		return err
	}

	if err := checkPlatformSupported(env); err != nil {
		return err
	}

	if err := checkClusterExists(ctx, client, env); err != nil {
		return err
	}

	return nil
}

// checkClusterExists checks if the cluster for the given environment is present in gcp.
func checkClusterExists(ctx context.Context, c *gcp.Client, env *environment.Environment) error {
	if _, err := c.GetCluster(ctx, env.Environment, env.Platform.(*environment.GCPVendor).Region, env.Platform.(*environment.GCPVendor).ProjectId); err != nil {
		return err
	}

	return nil
}

func checkPlatformSupported(env *environment.Environment) error {
	if env.Platform.Type() != environment.GCPVendorType {
		return fmt.Errorf("%s %w", strings.ToUpper(string(env.Platform.Type())), ErrCloudPlatformNotSupported)
	}
	return nil
}
