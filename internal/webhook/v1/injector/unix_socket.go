package injector

import corev1 "k8s.io/api/core/v1"

type UnixSocketInjector struct{}

func NewUnixSocketInjector() *UnixSocketInjector {
	// TODO: load from config
	// volumeName := "dfdaemon-socket"
	// volumePath := "/var/run/dragonfly/dfdaemon.sock"
	// volumeMountPath := "/var/run/dragonfly/dfdaemon.sock"
	// hostPathType := corev1.HostPathSocket
	// dfdaemonSocketVolume := corev1.Volume{
	// 	Name: volumeName,
	// 	VolumeSource: corev1.VolumeSource{
	// 		HostPath: &corev1.HostPathVolumeSource{
	// 			Path: volumePath,
	// 			Type: &hostPathType,
	// 		},
	// 	},
	// }
	// dfdaemonSocketVolumeMount := corev1.VolumeMount{
	// 	Name:      volumeName,
	// 	MountPath: volumeMountPath,
	// }
	// return &UnixSocketInjector{
	// 	DfdaemonSocketVolume:      &dfdaemonSocketVolume,
	// 	DfdaemonSocketVolumeMount: &dfdaemonSocketVolumeMount,
	// }
	return &UnixSocketInjector{}
}

func (usi *UnixSocketInjector) Inject(pod *corev1.Pod, config *InjectConf) {
	podlog.Info("UnixSocketInjector Inject")

	// check volume exsit
	volumeExsit := false
	for _, v := range pod.Spec.Volumes {
		if v.Name == DfdaemonUnixSockVolumeName {
			volumeExsit = true
			podlog.Info("volume exsit", "volume name", v.Name)
			break
		}
	}
	if !volumeExsit {
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
	// check volumeMount exsit
	exsit := false
	for _, v := range c.VolumeMounts {
		if v.Name == DfdaemonUnixSockVolumeName {
			exsit = true
			break
		}
	}
	if !exsit {
		dfdaemonSocketVolumeMount := corev1.VolumeMount{
			Name:      DfdaemonUnixSockVolumeName,
			MountPath: DfdaemonUnixSockPath,
		}
		c.VolumeMounts = append(c.VolumeMounts, dfdaemonSocketVolumeMount)
	}
}
