package injector

import (
	"os"
	"path/filepath"
	"sync"

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
	CliToolsDirPath           string = "/dragonfly-tools" // Cli tools binary directory path
)

type InjectConf struct {
	Enable    bool `json:"enable"`     // Whether to enable dragonfly injection
	ProxyPort int  `json:"proxy_port"` // Proxy port of dragonfly proxy(dfdaemon proxy port)
	// ProxyEnvValue   string `json:"proxy_env_value"` // Proxy url: "http://$(" + NodeNameEnvName + "):$(" + ProxyPortEnvName + ")"
	CliToolsImage   string `json:"cli_tools_image"`
	CliToolsDirPath string `json:"cli_tools_dir_path"`
}

func NewDefaultInjectConf() *InjectConf {
	return &InjectConf{
		Enable:    true,
		ProxyPort: ProxyPortEnvValue,
		// ProxyEnvValue:   fmt.Sprintf("http://$(%s):$(%s)", NodeNameEnvName, ProxyPortEnvName),
		CliToolsImage:   CliToolsImage,
		CliToolsDirPath: CliToolsDirPath,
	}
}

type ConfigManager struct {
	mu         sync.RWMutex
	config     *InjectConf
	configPath string
}

// TODO: Add config Manager
func NewConfigManager() *ConfigManager {
	configPath := filepath.Join(InjectConfigMapPath, "config.yaml")
	return &ConfigManager{
		mu:         sync.RWMutex{},
		config:     LoadInjectConf(configPath),
		configPath: configPath,
	}
}

func (cm *ConfigManager) GetConfig() *InjectConf {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.config
}

func (cm *ConfigManager) Watch() {
	return
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
	cfp := filepath.Join(injectConfigMapPath, "config.yaml")
	cf, err := os.ReadFile(cfp)
	if err != nil {
		return nil, err
	}
	injectConf := &InjectConf{}
	if err := yaml.Unmarshal(cf, injectConf); err != nil {
		return nil, err
	}

	return injectConf, nil
}
