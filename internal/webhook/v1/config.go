package v1

import (
	"os"

	corev1 "k8s.io/api/core/v1"
)

const (
	// ConfigMap Path
	InjectConfigMapPath string = "/etc/dragonfly-p2p-webhook/config"

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

type Injector interface {
	Inject(pod *corev1.Pod)
}

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

// load inject config from file
func LoadInjectConf() (*InjectConf, error) {
	dirs, err := os.ReadDir(InjectConfigMapPath)
	injectConf := &InjectConf{}
	if err != nil {
		podlog.Info(err.Error())
	}
	for _, dir := range dirs {
		podlog.Info(string(dir.Name()))
	}
	if err != nil {
		return nil, err
	}

	return injectConf, nil
}
