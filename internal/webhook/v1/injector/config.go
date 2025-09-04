package injector

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/util/yaml"
)

const (
	// ConfigMap Path, should config in config/default/manager_webhook_patch.yaml
	InjectConfigMapPath string = "/etc/dragonfly-p2p-webhook"

	// Namespace labels for injection control
	NamespaceInjectLabelName  string = "dragonflyoss-injection"
	NamespaceInjectLabelValue string = "enabled"

	// Pod annotation for injection control
	PodInjectAnnotationName  string = "dragonfly.io/inject"
	PodInjectAnnotationValue string = "true"

	// Environment variable control
	NodeNameEnvName   string = "NODE_NAME"
	ProxyPortEnvName  string = "DRAGONFLY_PROXY_PORT"
	ProxyPortEnvValue int    = 4001 // Default port of dragonfly proxy
	ProxyEnvName      string = "DRAGONFLY_INJECT_PROXY"

	// Dfdaemon unix sock volume control
	DfdaemonUnixSockVolumeName string = "dfdaemon-unix-sock"
	DfdaemonUnixSockPath       string = "/var/run/dragonfly/dfdaemon.sock" // Default path of dfdaemon unix sock

	// CliTools initContainer control
	CliToolsImageAnnotation   string = "dragonfly.io/cli-tools-image"  // Get specified cli tools image from this annotation
	CliToolsImage             string = "dragonflyoss/cli-tools:latest" // Default cli tools image
	CliToolsInitContainerName string = "d7y-cli-tools"
	CliToolsVolumeName        string = CliToolsInitContainerName + "-volume"
	CliToolsDirPath           string = "/dragonfly-tools"     // Cli tools binary directory path
	CliToolsPathEnvName       string = "DRAGONFLY_TOOLS_PATH" // Path to the directory where binaries are injected into the container.
)

type InjectConf struct {
	Enable          bool   `yaml:"enable" json:"enable"`         // Whether to enable dragonfly injection
	ProxyPort       int    `yaml:"proxy_port" json:"proxy_port"` // Proxy port of dragonfly proxy(dfdaemon proxy port)
	CliToolsImage   string `yaml:"cli_tools_image" json:"cli_tools_image"`
	CliToolsDirPath string `yaml:"cli_tools_dir_path" json:"cli_tools_dir_path"`
}

func NewDefaultInjectConf() *InjectConf {
	return &InjectConf{
		Enable:          true,
		ProxyPort:       ProxyPortEnvValue,
		CliToolsImage:   CliToolsImage,
		CliToolsDirPath: CliToolsDirPath,
	}
}

type ConfigManager struct {
	mu         sync.RWMutex
	config     *InjectConf
	configPath string
}

func NewConfigManager(injectConfigMapPath string) *ConfigManager {
	configPath := filepath.Join(injectConfigMapPath, "config.yaml")
	return &ConfigManager{
		mu:         sync.RWMutex{},
		config:     LoadInjectConf(configPath),
		configPath: configPath,
	}
}

func (cm *ConfigManager) GetConfig() *InjectConf {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	copiedConf := *cm.config
	podlog.Info("Get config", "config", copiedConf)
	return &copiedConf
}

func (cm *ConfigManager) Start(ctx context.Context) error {
	podlog.Info("Starting config file watcher.")

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			podlog.Info("Stopping config file watcher.")
			return nil
		case <-ticker.C:
			podlog.Info("Periodic reload check.")
			cm.reload()
		}
	}
}

func (cm *ConfigManager) reload() {
	config := LoadInjectConf(cm.configPath)
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.config = config
	podlog.Info("Configuration reloaded successfully.")
}

func LoadInjectConf(injectConfigMapPath string) *InjectConf {
	ic, err := LoadInjectConfFromFile(injectConfigMapPath)
	if err != nil {
		podlog.Error(err, "load config from file failed")
		podlog.Info("use default config")
		ic = NewDefaultInjectConf()
	}
	return ic
}

// load inject config from file
func LoadInjectConfFromFile(injectConfigMapPath string) (*InjectConf, error) {
	cf, err := os.ReadFile(injectConfigMapPath)
	if err != nil {
		return nil, err
	}
	injectConf := &InjectConf{}
	if err := yaml.Unmarshal(cf, injectConf); err != nil {
		return nil, err
	}

	return injectConf, nil
}
