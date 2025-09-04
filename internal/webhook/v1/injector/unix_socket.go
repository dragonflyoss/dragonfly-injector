package injector

import corev1 "k8s.io/api/core/v1"

type UnixSocketInjector struct{}

func NewUnixSocketInjector() *UnixSocketInjector {
	return &UnixSocketInjector{}
}

func (usi *UnixSocketInjector) Inject(pod *corev1.Pod, config *InjectConf) {
	podlog.Info("UnixSocketInjector Inject")

	// check volume exist
	volumeExist := false
	for _, v := range pod.Spec.Volumes {
		if v.Name == DfdaemonUnixSockVolumeName {
			volumeExist = true
			podlog.Info("volume exist", "volume name", v.Name)
			break
		}
	}
	if !volumeExist {
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
	for i := range pod.Spec.Containers {
		usi.InjectContainer(&pod.Spec.Containers[i])
	}
}

func (usi *UnixSocketInjector) InjectContainer(c *corev1.Container) {
	// check volumeMount exist
	exist := false
	for _, v := range c.VolumeMounts {
		if v.Name == DfdaemonUnixSockVolumeName {
			exist = true
			break
		}
	}
	if !exist {
		dfdaemonSocketVolumeMount := corev1.VolumeMount{
			Name:      DfdaemonUnixSockVolumeName,
			MountPath: DfdaemonUnixSockPath,
		}
		c.VolumeMounts = append(c.VolumeMounts, dfdaemonSocketVolumeMount)
	}
}
