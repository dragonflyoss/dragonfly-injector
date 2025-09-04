package injector

import (
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
)

type ToolsInitcontainerInjector struct{}

func NewToolsInitcontainerInjector() *ToolsInitcontainerInjector {
	return &ToolsInitcontainerInjector{}
}

func (tii *ToolsInitcontainerInjector) Inject(pod *corev1.Pod, config *InjectConf) {
	podlog.Info("ToolsInitcontainerInjector Inject")

	cliToolsVolumeMountPath := filepath.Clean(config.CliToolsDirPath) + "-mount"
	initContainerCmd := []string{
		"cp",
		"-rf",
		config.CliToolsDirPath + "/.",
		cliToolsVolumeMountPath + "/",
	}
	// get initContainerImage
	annotations := pod.Annotations
	initContainerImage := ""
	if annotations != nil {
		if _, ok := annotations[CliToolsImageAnnotation]; ok {
			initContainerImage = annotations[CliToolsImageAnnotation]
		} else {
			initContainerImage = config.CliToolsImage
		}
	}
	// add initContainer
	if !tii.CheckInitContainerIsExist(pod) {
		toolContainer := &corev1.Container{
			Name:            CliToolsInitContainerName,
			Image:           initContainerImage,
			ImagePullPolicy: corev1.PullIfNotPresent,
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      CliToolsVolumeName,
					MountPath: cliToolsVolumeMountPath,
				},
			},
			Command: initContainerCmd,
		}
		pod.Spec.InitContainers = append(pod.Spec.InitContainers, *toolContainer)
	}

	if !tii.CheckVolumeIsExist(pod) {
		toolsVolume := &corev1.Volume{
			Name: CliToolsVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		}
		pod.Spec.Volumes = append(pod.Spec.Volumes, *toolsVolume)
	}

	// add volumeMount and env
	for i := range pod.Spec.Containers {
		if !tii.CheckVolumeMountIsExist(&pod.Spec.Containers[i]) {
			pod.Spec.Containers[i].VolumeMounts = append(pod.Spec.Containers[i].VolumeMounts, corev1.VolumeMount{
				Name:      CliToolsVolumeName,
				MountPath: cliToolsVolumeMountPath,
			})
		}
		if !tii.CheckEnvIsExist(&pod.Spec.Containers[i]) {
			pod.Spec.Containers[i].Env = append(pod.Spec.Containers[i].Env, corev1.EnvVar{
				Name:  CliToolsPathEnvName,
				Value: cliToolsVolumeMountPath,
			})
		}
	}

}

// check initContainer is exist
func (tii *ToolsInitcontainerInjector) CheckInitContainerIsExist(pod *corev1.Pod) bool {
	ics := pod.Spec.InitContainers
	for i := range ics {
		if ics[i].Name == CliToolsInitContainerName {
			return true
		}
	}
	return false
}

// check volume is exist
func (tii *ToolsInitcontainerInjector) CheckVolumeIsExist(pod *corev1.Pod) bool {
	vs := pod.Spec.Volumes
	for i := range vs {
		if vs[i].Name == CliToolsVolumeName {
			return true
		}
	}
	return false
}

func (tii *ToolsInitcontainerInjector) CheckVolumeMountIsExist(c *corev1.Container) bool {
	for _, vm := range c.VolumeMounts {
		if vm.Name == CliToolsVolumeName {
			return true
		}
	}
	return false
}

// check cli tools path env is exist
func (tii *ToolsInitcontainerInjector) CheckEnvIsExist(c *corev1.Container) bool {
	env := c.Env
	for i := range env {
		if env[i].Name == CliToolsPathEnvName {
			return true
		}
	}
	return false
}
