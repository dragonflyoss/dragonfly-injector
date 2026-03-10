package injector

import (
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
)

type Tools struct{}

func NewTools() *Tools {
	return &Tools{}
}

func (t *Tools) Inject(pod *corev1.Pod, config *Config) {
	logger.Info("Tools inject", "pod namespace", pod.GetNamespace(), "pod name", pod.GetName())

	initContainerCmd := []string{
		"install",
		"-D",
		filepath.Join(config.CliToolsDirPath, CliToolsDfgetName),
		filepath.Join(config.CliToolsDirPath, CliToolsDfcacheName),
		filepath.Join(config.CliToolsDirPath, CliToolsDfstoreName),
		"-t",
		CliToolsVolumeMountPath + "/",
	}

	// Override initContainerImage with the value from annotations if it exists.
	annotations := pod.Annotations
	initContainerImage := config.CliToolsImage
	if annotations != nil {
		if image, ok := annotations[CliToolsImageAnnotation]; ok {
			initContainerImage = image
		}
	}

	// Mutate the pod spec to add the init container.
	if !t.hasInitContainer(pod) {
		toolContainer := &corev1.Container{
			Name:            CliToolsInitContainerName,
			Image:           initContainerImage,
			ImagePullPolicy: corev1.PullIfNotPresent,
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      CliToolsVolumeName,
					MountPath: CliToolsVolumeMountPath,
				},
			},
			Command: initContainerCmd,
		}
		pod.Spec.InitContainers = append(pod.Spec.InitContainers, *toolContainer)
	}

	if !t.hasVolume(pod) {
		toolsVolume := &corev1.Volume{
			Name: CliToolsVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		}
		pod.Spec.Volumes = append(pod.Spec.Volumes, *toolsVolume)
	}

	for i := range pod.Spec.Containers {
		if !t.hasVolumeMount(&pod.Spec.Containers[i]) {
			pod.Spec.Containers[i].VolumeMounts = append(pod.Spec.Containers[i].VolumeMounts, corev1.VolumeMount{
				Name:      CliToolsVolumeName,
				MountPath: CliToolsVolumeMountPath,
			})

			pod.Spec.Containers[i].VolumeMounts = append(pod.Spec.Containers[i].VolumeMounts, corev1.VolumeMount{
				Name:      CliToolsVolumeName,
				MountPath: filepath.Join(CliToolsMountDirPath, CliToolsDfgetName),
				SubPath:   CliToolsDfgetName,
				ReadOnly:  true,
			})

			pod.Spec.Containers[i].VolumeMounts = append(pod.Spec.Containers[i].VolumeMounts, corev1.VolumeMount{
				Name:      CliToolsVolumeName,
				MountPath: filepath.Join(CliToolsMountDirPath, CliToolsDfcacheName),
				SubPath:   CliToolsDfcacheName,
				ReadOnly:  true,
			})

			pod.Spec.Containers[i].VolumeMounts = append(pod.Spec.Containers[i].VolumeMounts, corev1.VolumeMount{
				Name:      CliToolsVolumeName,
				MountPath: filepath.Join(CliToolsMountDirPath, CliToolsDfstoreName),
				SubPath:   CliToolsDfstoreName,
				ReadOnly:  true,
			})
		}
	}
}

// hasInitContainer checks if the pod has the init container.
func (t *Tools) hasInitContainer(pod *corev1.Pod) bool {
	for _, c := range pod.Spec.InitContainers {
		if c.Name == CliToolsInitContainerName {
			return true
		}
	}

	return false
}

// hasVolume checks if the pod has the volume.
func (t *Tools) hasVolume(pod *corev1.Pod) bool {
	for _, v := range pod.Spec.Volumes {
		if v.Name == CliToolsVolumeName {
			return true
		}
	}

	return false
}

// hasVolumeMount checks if the container has the volume mount.
func (t *Tools) hasVolumeMount(c *corev1.Container) bool {
	for _, vm := range c.VolumeMounts {
		if vm.Name == CliToolsVolumeName {
			return true
		}
	}

	return false
}
