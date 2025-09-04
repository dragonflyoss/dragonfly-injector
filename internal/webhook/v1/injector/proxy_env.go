package injector

import (
	"strconv"

	corev1 "k8s.io/api/core/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var podlog = logf.Log.WithName("pod-resource")

type ProxyEnvInjector struct{}

func NewProxyEnvInjector() *ProxyEnvInjector {
	return &ProxyEnvInjector{}
}

func (pei *ProxyEnvInjector) Inject(pod *corev1.Pod, config *InjectConf) {
	podlog.Info("ProxyEnvInjector Inject")

	envs := envsFromConfig(config)
	// inject env to all containers
	containers := pod.Spec.Containers
	for i := range containers {
		injectContainer(&containers[i], envs)
	}
}

func envsFromConfig(config *InjectConf) []corev1.EnvVar {
	envs := []corev1.EnvVar{
		{
			Name: NodeNameEnvName,
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "spec.nodeName",
				},
			},
		},
		{
			Name:  ProxyPortEnvName,
			Value: strconv.Itoa(config.ProxyPort),
		},
		{
			Name:  ProxyEnvName,
			Value: "http://$(" + NodeNameEnvName + "):$(" + ProxyPortEnvName + ")",
		},
	}
	return envs
}
func injectContainer(c *corev1.Container, envs []corev1.EnvVar) {
	for _, e := range envs {
		exsit := false
		// if env exsit, skip
		for _, ce := range c.Env {
			if e.Name == ce.Name {
				exsit = true
				break
			}
		}
		if !exsit {
			c.Env = append(c.Env, e)
		}
	}
}
