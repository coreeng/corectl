package tenant

import (
	"gopkg.in/yaml.v3"
	"io/fs"
	"os"
	"path/filepath"
)

func List(cplatformRepo string) ([]Tenant, error) {
	iter := tenantsIterator(cplatformRepo)
	var tenants []Tenant
	for {
		t, done, err := iter()
		if err != nil {
			return nil, err
		}
		if done {
			break
		}
		tenants = append(tenants, t)
	}
	return tenants, nil
}

func FindByName(cplatformRepo string, name Name) (*Tenant, error) {
	iter := tenantsIterator(cplatformRepo)
	for {
		t, done, err := iter()
		if err != nil {
			return nil, err
		}
		if done {
			return nil, nil
		}
		if t.Name == name {
			return &t, nil
		}
	}
}

func tenantsIterator(configRepoPath string) func() (Tenant, bool, error) {
	tenantsCh := make(chan Tenant)
	errCh := make(chan error)
	go func() {
		tenantsPath := filepath.Join(configRepoPath, tenantsRelativePath)
		if err := fs.WalkDir(os.DirFS(tenantsPath), ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || filepath.Ext(d.Name()) != ".yaml" {
				return nil
			}
			tenantBytes, err := os.ReadFile(filepath.Join(tenantsPath, path))
			if err != nil {
				return err
			}
			var tenant Tenant
			if err = yaml.Unmarshal(tenantBytes, &tenant); err != nil {
				return err
			}
			tenant.path = &path
			tenantsCh <- tenant
			return nil
		}); err != nil {
			errCh <- err
		}
		close(tenantsCh)
		close(errCh)
	}()

	return func() (Tenant, bool, error) {
		select {
		case tenant, isReceived := <-tenantsCh:
			return tenant, !isReceived, nil
		case err, isReceived := <-errCh:
			return Tenant{}, !isReceived, err
		}
	}
}
