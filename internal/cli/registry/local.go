package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	RegistryDir     = ".omdr"
	RegistryFile    = "servers.yaml"
	DefaultRegistry = "local"
)

// ServerConfig represents the execution configuration for a server
type ServerConfig struct {
	Command   string            `yaml:"command"`
	Args      []string          `yaml:"args,omitempty"`
	Env       map[string]string `yaml:"env,omitempty"`
	Secrets   []string          `yaml:"secrets,omitempty"` // List of secret keys to fetch from keychain
	UpdatedAt time.Time         `yaml:"updated_at"`
}

// LocalRegistry manages the local database of installed servers
type LocalRegistry struct {
	path string
	mu   sync.RWMutex
}

// NewLocalRegistry creates a new registry manager using user home
func NewLocalRegistry() (*LocalRegistry, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("getting home directory: %w", err)
	}
	return NewLocalRegistryWithHome(home)
}

// NewLocalRegistryWithHome creates a new registry manager with explicit home
func NewLocalRegistryWithHome(homeDir string) (*LocalRegistry, error) {
	path := filepath.Join(homeDir, RegistryDir, RegistryFile)
	return &LocalRegistry{
		path: path,
	}, nil
}

type registryData struct {
	Servers map[string]ServerConfig `yaml:"servers"`
}

// Register adds or updates a server configuration
func (r *LocalRegistry) Register(serverName string, config ServerConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := r.load()
	if err != nil {
		return err
	}

	config.UpdatedAt = time.Now()
	data.Servers[serverName] = config

	return r.save(data)
}

// Get retrieves a server configuration
func (r *LocalRegistry) Get(serverName string) (*ServerConfig, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	data, err := r.load()
	if err != nil {
		return nil, err
	}

	config, ok := data.Servers[serverName]
	if !ok {
		return nil, fmt.Errorf("server not found: %s", serverName)
	}

	return &config, nil
}

// Unregister removes a server configuration
func (r *LocalRegistry) Unregister(serverName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := r.load()
	if err != nil {
		return err
	}

	delete(data.Servers, serverName)

	return r.save(data)
}

// List returns all registered servers
func (r *LocalRegistry) List() (map[string]ServerConfig, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	data, err := r.load()
	if err != nil {
		return nil, err
	}

	return data.Servers, nil
}

func (r *LocalRegistry) load() (*registryData, error) {
	data := &registryData{
		Servers: make(map[string]ServerConfig),
	}

	// Check if file exists
	if _, err := os.Stat(r.path); os.IsNotExist(err) {
		return data, nil
	}

	content, err := os.ReadFile(r.path)
	if err != nil {
		return nil, fmt.Errorf("reading registry file: %w", err)
	}

	if len(content) == 0 {
		return data, nil
	}

	if err := yaml.Unmarshal(content, data); err != nil {
		return nil, fmt.Errorf("parsing registry file: %w", err)
	}

	if data.Servers == nil {
		data.Servers = make(map[string]ServerConfig)
	}

	return data, nil
}

func (r *LocalRegistry) save(data *registryData) error {
	// Ensure directory exists
	dir := filepath.Dir(r.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating registry directory: %w", err)
	}

	content, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshaling registry data: %w", err)
	}

	if err := os.WriteFile(r.path, content, 0644); err != nil {
		return fmt.Errorf("writing registry file: %w", err)
	}

	return nil
}
