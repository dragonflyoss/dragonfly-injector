package injector

import (
	"time"
)

const (
	// ConfigReloadWaitTime is wait time for config reload.
	ConfigReloadWaitTime time.Duration = 30 * time.Second

	// Domain is the domain name of the injector.
	Domain string = "dragonfly.io"

	// ConfigMap Path, should config in config/default/manager_webhook_patch.yaml.
	ConfigMapPath string = "/etc/dragonfly-injector"

	// Namespace labels for injection control.
	NamespaceInjectLabelName  string = Domain + "/" + "inject"
	NamespaceInjectLabelValue string = "true"

	// Pod annotation for injection control.
	PodInjectAnnotationName  string = Domain + "/" + "inject"
	PodInjectAnnotationValue string = "true"

	// Dfdaemon unix sock config.
	DfdaemonUnixSockVolumeName string = "dfdaemon-unix-sock"
	DfdaemonUnixSockPath       string = "/var/run/dragonfly/dfdaemon.sock" // Default path of dfdaemon unix sock

	// CliTools initContainer control.
	CliToolsImage           string = "dragonflyoss/client:latest"     // Default cli tools image
	CliToolsImageAnnotation string = Domain + "/" + "cli-tools-image" // Get specified cli tools image from this annotation

	CliToolsInitContainerName string = "d7y-cli-tools"
	CliToolsVolumeName        string = CliToolsInitContainerName + "-" + "volume"
	CliToolsVolumeMountPath   string = "/d7y/bin"
	CliToolsDirPath           string = "/usr/local/bin" // Default cli tools binary directory path in cli tools image
	CliToolsMountDirPath      string = "/usr/local/bin"

	CliToolsDfgetName   string = "dfget"
	CliToolsDfcacheName string = "dfcache"
	CliToolsDfstoreName string = "dfstore"
)
