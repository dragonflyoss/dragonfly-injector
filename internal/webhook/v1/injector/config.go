package injector

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var logger = log.Log.WithName("dragonfly-inject")

// Config is the configuration for dragonfly injection.
type Config struct {
	// Whether to enable dragonfly injection function.
	Enable bool `yaml:"enable" json:"enable"`

	// CliToolsImage is the image of the cli tools container.
	CliToolsImage string `yaml:"cli_tools_image" json:"cli_tools_image"`

	// CliToolsDirPath is the directory path where the cli tools are located.
	CliToolsDirPath string `yaml:"cli_tools_dir_path" json:"cli_tools_dir_path"`
}

func DefaultConfig() *Config {
	return &Config{
		Enable:          true,
		CliToolsImage:   CliToolsImage,
		CliToolsDirPath: CliToolsDirPath,
	}
}

type ConfigManager struct {
	mu         sync.RWMutex
	config     *Config
	configPath string
}

func NewConfigManager(injectConfigMapPath string) *ConfigManager {
	configPath := filepath.Join(injectConfigMapPath, "config.yaml")
	return &ConfigManager{
		mu:         sync.RWMutex{},
		config:     LoadConfig(configPath),
		configPath: configPath,
	}
}

func (cm *ConfigManager) GetConfig() *Config {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	copiedConf := *cm.config
	logger.Info("Get config", "config", copiedConf)
	return &copiedConf
}

func (cm *ConfigManager) Start(ctx context.Context) error {
	logger.Info("Starting config file watcher")

	ticker := time.NewTicker(ConfigReloadWaitTime)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			logger.Info("Stopping config file watcher")
			return nil
		case <-ticker.C:
			logger.Info("Periodic reload check")
			cm.reload()
		}
	}
}

func (cm *ConfigManager) reload() {
	config := LoadConfig(cm.configPath)
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.config = config
	logger.Info("Configuration reloaded successfully")
}

func LoadConfig(injectConfigMapPath string) *Config {
	c, err := LoadConfigFromFile(injectConfigMapPath)
	if err != nil {
		logger.Error(err, "load config from file failed")
		logger.Info("use default config")
		c = DefaultConfig()
	}
	return c
}

// LoadConfigFromFile loads config from file.
func LoadConfigFromFile(injectConfigMapPath string) (*Config, error) {
	cf, err := os.ReadFile(injectConfigMapPath)
	if err != nil {
		return nil, err
	}

	config := &Config{}
	if err := yaml.Unmarshal(cf, config); err != nil {
		return nil, err
	}

	return config, nil
}
