package injector

import (
	corev1 "k8s.io/api/core/v1"
)

type ToolsInitcontainerInjector struct {
	// dragonfly tools initcontainer name
	initContainerName string

	// dragonfly tools initcontainer image annotation
	initContainerImageAnnotation string

	// dragonfly tools binary dir
	initContainerToolsDir string

	// dragonfly tools initcontainer volume mount path
	// initContainerVolumeMountPath string

	// dragonfly tools initcontainer command
	initContainerCommand      []string
	defaultInitContainerImage string
	toolsVolumeName           string
	toolsVolumeMountPath      string
	dragonflyToolsPathEnvName string
}

func NewToolsInitcontainerInjector() *ToolsInitcontainerInjector {
	initContainerToolsDir := "/dragonfly-tools"
	initContainerVolumeMountPath := "/dragonfly-tools-mount"
	return &ToolsInitcontainerInjector{
		initContainerName:            "dragonfly-tools",
		initContainerImageAnnotation: "dragonfly.io/cli-tools-image",
		defaultInitContainerImage:    "dragonflyoss/cli-tools:latest",
		initContainerToolsDir:        initContainerToolsDir,
		// initContainerVolumeMountPath: initContainerVolumeMountPath,
		initContainerCommand: []string{
			"cp",
			"-rf",
			initContainerToolsDir + "/.",
			initContainerVolumeMountPath + "/",
		},
		toolsVolumeName:           "dragonfly-tools-volume",
		toolsVolumeMountPath:      initContainerVolumeMountPath,
		dragonflyToolsPathEnvName: "DRAGONFLY_TOOLS_PATH",
	}
}

func (tii *ToolsInitcontainerInjector) Inject(pod *corev1.Pod) {
	podlog.Info("ToolsInitcontainerInjector Inject")
	// get initContainerImage
	annotations := pod.Annotations
	initContainerImage := ""
	if annotations != nil {
		if _, ok := annotations[tii.initContainerImageAnnotation]; ok {
			initContainerImage = annotations[tii.initContainerImageAnnotation]
		} else {
			initContainerImage = tii.defaultInitContainerImage
		}
	}
	// add initContainer
	if !tii.CheckInitContainerIsExist(pod) {
		toolContainer := &corev1.Container{
			Name:            tii.initContainerName,
			Image:           initContainerImage,
			ImagePullPolicy: corev1.PullIfNotPresent,
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      tii.toolsVolumeName,
					MountPath: tii.toolsVolumeMountPath,
				},
			},
			Command: tii.initContainerCommand,
		}
		pod.Spec.InitContainers = append(pod.Spec.InitContainers, *toolContainer)
	}

	if !tii.CheckVolumeIsExist(pod) {
		toolsVolume := &corev1.Volume{
			Name: tii.toolsVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		}
		pod.Spec.Volumes = append(pod.Spec.Volumes, *toolsVolume)
	}

	// add volumeMount and env
	for i := range pod.Spec.Containers {
		if !tii.CheckVolumeMountIsExist(pod) {
			pod.Spec.Containers[i].VolumeMounts = append(pod.Spec.Containers[i].VolumeMounts, corev1.VolumeMount{
				Name:      tii.toolsVolumeName,
				MountPath: tii.toolsVolumeMountPath,
			})
		}
		if !tii.CheckEnvIsExist(&pod.Spec.Containers[i]) {
			pod.Spec.Containers[i].Env = append(pod.Spec.Containers[i].Env, corev1.EnvVar{
				Name:  tii.dragonflyToolsPathEnvName,
				Value: tii.toolsVolumeMountPath,
			})
		}
	}

}

// check initContainer is exist
func (tii *ToolsInitcontainerInjector) CheckInitContainerIsExist(pod *corev1.Pod) bool {
	ics := pod.Spec.InitContainers
	for i := range ics {
		if ics[i].Name == tii.initContainerName {
			return true
		}
	}
	return false
}

// check volume is exist
func (tii *ToolsInitcontainerInjector) CheckVolumeIsExist(pod *corev1.Pod) bool {
	vs := pod.Spec.Volumes
	for i := range vs {
		if vs[i].Name == tii.toolsVolumeName {
			return true
		}
	}
	return false
}

func (tii *ToolsInitcontainerInjector) CheckVolumeMountIsExist(pod *corev1.Pod) bool {
	vm := pod.Spec.Containers[0].VolumeMounts
	for i := range vm {
		if vm[i].Name == tii.toolsVolumeName {
			return true
		}
	}
	return false
}

// check volumeMountPath is in env PATH
func (tii *ToolsInitcontainerInjector) CheckEnvIsExist(c *corev1.Container) bool {
	env := c.Env
	for i := range env {
		if env[i].Name == tii.dragonflyToolsPathEnvName {
			return true
		}
	}
	return false
}
