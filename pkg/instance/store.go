package instance

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/coreeng/corectl/pkg/cmdutil/configpath"
)

const (
	BuiltInName   = "core-platform"
	BuiltInOrigin = "https://portal.coreplatform.io"
)

type Instance struct {
	Name   string `json:"name"`
	Origin string `json:"origin"`
}

type Binding struct {
	Generation     string `json:"generation"`
	ManagedContext string `json:"managedContext"`
}

type data struct {
	Current   string             `json:"current,omitempty"`
	Instances map[string]string  `json:"instances,omitempty"`
	Bindings  map[string]Binding `json:"bindings,omitempty"`
}

type Store struct {
	Path string
}

func DefaultStore() *Store {
	return &Store{Path: filepath.Join(configpath.GetCorectlHomeDir(), "platform.json")}
}

func NormalizeOrigin(raw string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("invalid instance URL %q", raw)
	}
	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)
	if (u.Scheme == "https" && u.Port() == "443") || (u.Scheme == "http" && u.Port() == "80") {
		u.Host = u.Hostname()
		if strings.Contains(u.Host, ":") {
			u.Host = net.JoinHostPort(u.Host, "")
			u.Host = strings.TrimSuffix(u.Host, ":")
		}
	}
	if u.Scheme != "https" {
		if u.Scheme != "http" || (u.Hostname() != "localhost" && u.Hostname() != "127.0.0.1") {
			return "", errors.New("instance URL must use HTTPS (HTTP is allowed for localhost)")
		}
	}
	if u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return "", errors.New("instance URL must be an origin without a path, query, fragment, or credentials")
	}
	u.Path = ""
	return u.String(), nil
}

func (s *Store) List() ([]Instance, error) {
	d, err := s.load()
	if err != nil {
		return nil, err
	}
	result := []Instance{{Name: BuiltInName, Origin: BuiltInOrigin}}
	for name, origin := range d.Instances {
		result = append(result, Instance{Name: name, Origin: origin})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

func (s *Store) Resolve(name string) (Instance, error) {
	d, err := s.load()
	if err != nil {
		return Instance{}, err
	}
	if name == "" {
		name = d.Current
		if name == "" {
			name = BuiltInName
		}
	}
	if name == BuiltInName {
		return Instance{Name: name, Origin: BuiltInOrigin}, nil
	}
	origin, ok := d.Instances[name]
	if !ok {
		return Instance{}, fmt.Errorf("unknown instance %q", name)
	}
	return Instance{Name: name, Origin: origin}, nil
}

func (s *Store) Add(name, rawOrigin string) error {
	if name == "" || name == BuiltInName || strings.ContainsAny(name, " /\\") {
		return fmt.Errorf("invalid custom instance name %q", name)
	}
	origin, err := NormalizeOrigin(rawOrigin)
	if err != nil {
		return err
	}
	d, err := s.load()
	if err != nil {
		return err
	}
	if _, exists := d.Instances[name]; exists {
		return fmt.Errorf("instance %q already exists", name)
	}
	d.Instances[name] = origin
	return s.save(d)
}

func (s *Store) Use(name string) error {
	if _, err := s.Resolve(name); err != nil {
		return err
	}
	d, err := s.load()
	if err != nil {
		return err
	}
	d.Current = name
	return s.save(d)
}

func (s *Store) Remove(name string) error {
	if name == BuiltInName {
		return errors.New("the built-in core-platform instance cannot be removed")
	}
	d, err := s.load()
	if err != nil {
		return err
	}
	if _, ok := d.Instances[name]; !ok {
		return fmt.Errorf("unknown instance %q", name)
	}
	origin := d.Instances[name]
	delete(d.Instances, name)
	for key := range d.Bindings {
		if strings.HasPrefix(key, origin+"\n") {
			delete(d.Bindings, key)
		}
	}
	if d.Current == name {
		d.Current = BuiltInName
	}
	return s.save(d)
}

func (s *Store) Binding(origin, clusterID string) (Binding, bool, error) {
	d, err := s.load()
	if err != nil {
		return Binding{}, false, err
	}
	b, ok := d.Bindings[bindingKey(origin, clusterID)]
	return b, ok, nil
}

func (s *Store) SetBinding(origin, clusterID string, binding Binding) error {
	d, err := s.load()
	if err != nil {
		return err
	}
	d.Bindings[bindingKey(origin, clusterID)] = binding
	return s.save(d)
}

func bindingKey(origin, clusterID string) string { return origin + "\n" + clusterID }

func (s *Store) load() (data, error) {
	d := data{Instances: map[string]string{}, Bindings: map[string]Binding{}}
	b, err := os.ReadFile(s.Path)
	if errors.Is(err, os.ErrNotExist) {
		return d, nil
	}
	if err != nil {
		return d, fmt.Errorf("read Corectl platform state: %w", err)
	}
	if err := json.Unmarshal(b, &d); err != nil {
		return d, fmt.Errorf("decode Corectl platform state: %w", err)
	}
	if d.Instances == nil {
		d.Instances = map[string]string{}
	}
	if d.Bindings == nil {
		d.Bindings = map[string]Binding{}
	}
	return d, nil
}

func (s *Store) save(d data) error {
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o700); err != nil {
		return fmt.Errorf("create Corectl config directory: %w", err)
	}
	b, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.Path + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o600); err != nil {
		return fmt.Errorf("write Corectl platform state: %w", err)
	}
	if err := os.Rename(tmp, s.Path); err != nil {
		return fmt.Errorf("replace Corectl platform state: %w", err)
	}
	return nil
}
