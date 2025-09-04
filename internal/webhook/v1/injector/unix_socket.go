package injector

import corev1 "k8s.io/api/core/v1"

type UnixSocketInjector struct {
	DfdaemonSocketVolume      *corev1.Volume
	DfdaemonSocketVolumeMount *corev1.VolumeMount
}

func NewUnixSocketInjector() *UnixSocketInjector {
	// TODO: load from config
	volumeName := "dfdaemon-socket"
	volumePath := "/var/run/dragonfly/dfdaemon.sock"
	volumeMountPath := "/var/run/dragonfly/dfdaemon.sock"
	hostPathType := corev1.HostPathSocket
	dfdaemonSocketVolume := corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: volumePath,
				Type: &hostPathType,
			},
		},
	}
	dfdaemonSocketVolumeMount := corev1.VolumeMount{
		Name:      volumeName,
		MountPath: volumeMountPath,
	}
	return &UnixSocketInjector{
		DfdaemonSocketVolume:      &dfdaemonSocketVolume,
		DfdaemonSocketVolumeMount: &dfdaemonSocketVolumeMount,
	}
}

func (usi *UnixSocketInjector) Inject(pod *corev1.Pod) {
	podlog.Info("UnixSocketInjector Inject")

	// check volume exsit
	volumeExsit := false
	for _, v := range pod.Spec.Volumes {
		if v.Name == usi.DfdaemonSocketVolume.Name {
			volumeExsit = true
			podlog.Info("volume exsit", "volume name", v.Name)
			break
		}
	}
	if !volumeExsit {
		pod.Spec.Volumes = append(pod.Spec.Volumes, *usi.DfdaemonSocketVolume)
	}
	// TODO: if volume exsist, load this volume conf to inject
	for i := range pod.Spec.Containers {
		usi.InjectContainer(&pod.Spec.Containers[i])
	}
}

func (usi *UnixSocketInjector) InjectContainer(c *corev1.Container) {
	// check volumeMount exsit
	exsit := false
	for _, v := range c.VolumeMounts {
		if v.Name == usi.DfdaemonSocketVolumeMount.Name {
			exsit = true
			break
		}
	}
	if !exsit {
		c.VolumeMounts = append(c.VolumeMounts, *usi.DfdaemonSocketVolumeMount)
	}
}
