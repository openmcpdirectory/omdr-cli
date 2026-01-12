package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

const (
	GlobalConfigDir  = ".omdr"
	GlobalConfigFile = "config.yaml"
	LocalConfigFile  = "omdr.yaml"
)

type Manager struct {
	globalPath string
	localPath  string
}

func NewManager() (*Manager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("getting home directory: %w", err)
	}

	globalPath := filepath.Join(home, GlobalConfigDir, GlobalConfigFile)
	localPath := LocalConfigFile

	return &Manager{
		globalPath: globalPath,
		localPath:  localPath,
	}, nil
}

func (m *Manager) Get(key string) (string, error) {
	if err := m.loadConfigs(); err != nil {
		return "", err
	}

	value := viper.GetString(key)
	if value == "" {
		return "", fmt.Errorf("key not found: %s", key)
	}

	return value, nil
}

func (m *Manager) Set(key, value string) error {
	configPath := m.globalPath

	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	var config map[string]interface{}

	data, err := os.ReadFile(configPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading config file: %w", err)
	}

	if len(data) > 0 {
		if err := yaml.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("parsing config file: %w", err)
		}
	} else {
		config = make(map[string]interface{})
	}

	config[key] = value

	output, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(configPath, output, 0644); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	return nil
}

func (m *Manager) loadConfigs() error {
	viper.Reset()

	if _, err := os.Stat(m.globalPath); err == nil {
		viper.SetConfigFile(m.globalPath)
		if err := viper.ReadInConfig(); err != nil {
			return fmt.Errorf("reading global config: %w", err)
		}
	}

	if _, err := os.Stat(m.localPath); err == nil {
		viper.SetConfigFile(m.localPath)
		if err := viper.MergeInConfig(); err != nil {
			return fmt.Errorf("merging local config: %w", err)
		}
	}

	return nil
}

func (m *Manager) GetGlobalPath() string {
	return m.globalPath
}

func (m *Manager) GetLocalPath() string {
	return m.localPath
}
