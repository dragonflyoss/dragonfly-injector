package injector

import (
	corev1 "k8s.io/api/core/v1"
)

type UnixSocket struct{}

func NewUnixSocket() *UnixSocket {
	return &UnixSocket{}
}

func (us *UnixSocket) Inject(pod *corev1.Pod, config *Config) {
	logger.Info("UnixSocket inject", "pod namespace", pod.GetNamespace(), "pod name", pod.GetName())

	if !us.hasVolume(pod) {
		hostPathType := corev1.HostPathSocket
		dfdaemonSocketVolume := corev1.Volume{
			Name: DfdaemonUnixSockVolumeName,
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: DfdaemonUnixSockPath,
					Type: &hostPathType,
				},
			},
		}
		pod.Spec.Volumes = append(pod.Spec.Volumes, dfdaemonSocketVolume)
	}

	for i, c := range pod.Spec.Containers {
		if !us.hasVolumeMount(&c) {
			dfdaemonSocketVolumeMount := corev1.VolumeMount{
				Name:      DfdaemonUnixSockVolumeName,
				MountPath: DfdaemonUnixSockPath,
			}
			pod.Spec.Containers[i].VolumeMounts = append(pod.Spec.Containers[i].VolumeMounts, dfdaemonSocketVolumeMount)
		}
	}
}

// hasVolume checks if the pod has the volume.
func (us *UnixSocket) hasVolume(pod *corev1.Pod) bool {
	for _, v := range pod.Spec.Volumes {
		if v.Name == DfdaemonUnixSockVolumeName {
			return true
		}
	}

	return false
}

// hasVolumeMount checks if the container has the volume mount.
func (us *UnixSocket) hasVolumeMount(c *corev1.Container) bool {
	for _, vm := range c.VolumeMounts {
		if vm.Name == DfdaemonUnixSockVolumeName {
			return true
		}
	}

	return false
}
