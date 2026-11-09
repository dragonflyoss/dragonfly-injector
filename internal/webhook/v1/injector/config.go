/*
 *     Copyright 2026 The Dragonfly Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package injector

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var logger = log.Log.WithName("dragonfly-inject")

// Config is the configuration for dragonfly injection.
type Config struct {
	InitContainerImage InitContainerImage `yaml:"initContainerImage" json:"initContainerImage"`
}

// InitContainerImage is the image configuration for the init container.
type InitContainerImage struct {
	Registry    string                        `yaml:"registry" json:"registry"`
	Repository  string                        `yaml:"repository" json:"repository"`
	Tag         string                        `yaml:"tag" json:"tag"`
	Digest      string                        `yaml:"digest" json:"digest"`
	PullPolicy  corev1.PullPolicy             `yaml:"pullPolicy" json:"pullPolicy"`
	PullSecrets []corev1.LocalObjectReference `yaml:"pullSecrets" json:"pullSecrets"`
}

// GetInitContainerImageReference returns the init container image reference.
func (c *Config) GetInitContainerImageReference() string {
	reference := c.InitContainerImage.Registry + "/" + c.InitContainerImage.Repository
	if c.InitContainerImage.Tag != "" {
		reference += ":" + c.InitContainerImage.Tag
	}
	if c.InitContainerImage.Digest != "" {
		reference += "@" + c.InitContainerImage.Digest
	}

	return reference
}

func DefaultConfig() *Config {
	return &Config{
		InitContainerImage: InitContainerImage{
			Registry:    "docker.io",
			Repository:  "dragonflyoss/client",
			Tag:         "v1.3.0",
			PullPolicy:  "IfNotPresent",
			PullSecrets: []corev1.LocalObjectReference{},
		},
	}
}

type ConfigManager struct {
	mu         sync.RWMutex
	config     *Config
	configPath string
	lastRaw    []byte
}

func NewConfigManager(injectConfigMapPath string) *ConfigManager {
	configPath := filepath.Join(injectConfigMapPath, "injector.yaml")
	raw, err := os.ReadFile(configPath)
	if err != nil {
		logger.Error(err, "load config from file failed")
		logger.Info("use default config")
		return &ConfigManager{config: DefaultConfig(), configPath: configPath}
	}

	return &ConfigManager{
		config:     loadConfigFromBytes(raw),
		configPath: configPath,
		lastRaw:    raw,
	}
}

func (cm *ConfigManager) GetConfig() *Config {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return cm.config.DeepCopy()
}

// DeepCopy returns a deep copy of the config, so callers cannot mutate
// shared fields (e.g. PullSecrets) of the cached config through the
// returned copy.
func (c *Config) DeepCopy() *Config {
	if c == nil {
		return nil
	}

	copiedConf := *c
	if c.InitContainerImage.PullSecrets != nil {
		copiedConf.InitContainerImage.PullSecrets = make([]corev1.LocalObjectReference, len(c.InitContainerImage.PullSecrets))
		copy(copiedConf.InitContainerImage.PullSecrets, c.InitContainerImage.PullSecrets)
	}
	return &copiedConf
}

func (cm *ConfigManager) Start(ctx context.Context) error {
	logger.Info("Starting config file watcher")

	ticker := time.NewTicker(ConfigReloadInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			logger.Info("Stopping config file watcher")
			return nil
		case <-ticker.C:
			cm.reload()
		}
	}
}

func (cm *ConfigManager) reload() {
	raw, err := os.ReadFile(cm.configPath)
	if err != nil {
		logger.Error(err, "read config file for reload", "path", cm.configPath)
		return
	}

	if bytes.Equal(raw, cm.lastRaw) {
		return
	}

	config, err := parseConfig(raw)
	if err != nil {
		logger.Error(err, "parse config file during reload", "path", cm.configPath)
		return
	}

	if err := config.Validate(); err != nil {
		logger.Error(err, "invalid config on reload, keeping current")
		return
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.config = config
	cm.lastRaw = raw
	logger.Info("Configuration reloaded")
}

func LoadConfig(injectConfigMapPath string) *Config {
	raw, err := os.ReadFile(injectConfigMapPath)
	if err != nil {
		logger.Error(err, "load config from file failed")
		logger.Info("use default config")
		return DefaultConfig()
	}

	return loadConfigFromBytes(raw)
}

// loadConfigFromBytes parses raw config bytes, falling back to the default
// config if parsing or validation fails.
func loadConfigFromBytes(raw []byte) *Config {
	c, err := parseConfig(raw)
	if err != nil {
		logger.Error(err, "load config from file failed")
		logger.Info("use default config")
		return DefaultConfig()
	}

	if err := c.Validate(); err != nil {
		logger.Error(err, "invalid config, using defaults")
		return DefaultConfig()
	}

	return c
}

// LoadConfigFromFile loads config from file. Fields not present in the
// file keep their default values, so a partial config file (e.g. only
// overriding the image tag) does not silently reset the rest of the
// configuration to zero values.
func LoadConfigFromFile(injectConfigMapPath string) (*Config, error) {
	raw, err := os.ReadFile(injectConfigMapPath)
	if err != nil {
		return nil, err
	}

	return parseConfig(raw)
}

// parseConfig unmarshals raw YAML bytes into a Config pre-populated
// with the default values. Fields not present in the raw bytes keep
// their defaults, so a partial config file (e.g. only overriding the
// image tag) does not silently reset the rest of the configuration to
// zero values.
func parseConfig(raw []byte) (*Config, error) {
	config := DefaultConfig()
	if err := yaml.Unmarshal(raw, config); err != nil {
		return nil, err
	}

	return config, nil
}

// Validate checks that required fields are set.
func (c *Config) Validate() error {
	if c.InitContainerImage.Registry == "" {
		return fmt.Errorf("initContainerImage.registry is required")
	}
	if c.InitContainerImage.Repository == "" {
		return fmt.Errorf("initContainerImage.repository is required")
	}
	return nil
}
