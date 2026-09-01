package platformruntime

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/coreeng/corectl/pkg/auth"
	clusterpkg "github.com/coreeng/corectl/pkg/cluster"
	"github.com/coreeng/corectl/pkg/instance"
	"github.com/coreeng/corectl/pkg/portal"
	"github.com/pkg/browser"
	"github.com/zalando/go-keyring"
)

type ClusterConnector interface {
	Connect(context.Context, instance.Instance, portal.Cluster, string, bool) (string, error)
}

type Runtime struct {
	Instances    *instance.Store
	Tokens       auth.Store
	HTTPClient   *http.Client
	OpenURL      func(string) error
	Sleep        func(context.Context, time.Duration) error
	Connector    ClusterConnector
	Installer    clusterpkg.Installer
	InstanceName string
}

func Default() *Runtime {
	instances := instance.DefaultStore()
	return &Runtime{
		Instances:  instances,
		Tokens:     auth.KeyringStore{},
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
		OpenURL:    browser.OpenURL,
		Sleep: func(ctx context.Context, duration time.Duration) error {
			timer := time.NewTimer(duration)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
				return nil
			}
		},
		Connector: clusterpkg.NewConnector(instances),
		Installer: clusterpkg.HelmInstaller{},
	}
}

func (r *Runtime) Selected() (instance.Instance, error) {
	name := r.InstanceName
	if name == "" {
		name = os.Getenv("CORECTL_INSTANCE")
	}
	return r.Instances.Resolve(name)
}

func (r *Runtime) AuthenticatedClient() (*portal.Client, instance.Instance, error) {
	selected, err := r.Selected()
	if err != nil {
		return nil, instance.Instance{}, err
	}
	token, err := r.Tokens.Get(selected.Origin)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return nil, instance.Instance{}, fmt.Errorf("not logged in to %s; run corectl login", selected.Name)
		}
		return nil, instance.Instance{}, fmt.Errorf("read token from OS keychain: %w", err)
	}
	if !token.ExpiresAt.IsZero() && time.Now().After(token.ExpiresAt) {
		return nil, instance.Instance{}, fmt.Errorf("login for %s has expired; run corectl login", selected.Name)
	}
	return portal.New(selected.Origin, r.HTTPClient, token.AccessToken), selected, nil
}
