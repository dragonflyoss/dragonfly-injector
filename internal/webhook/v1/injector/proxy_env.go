package injector

import (
	corev1 "k8s.io/api/core/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var podlog = logf.Log.WithName("pod-resource")

type ProxyEnvInjector struct {
	Envs []corev1.EnvVar
}

func NewProxyEnvInjector() *ProxyEnvInjector {
	return &ProxyEnvInjector{
		Envs: []corev1.EnvVar{
			{
				Name: "NODE_NAME",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "spec.nodeName",
					},
				},
			},
			{
				Name: "DRAGONFLY_PROXY_PORT",
				// TODO: get proxy port from helm yaml
				Value: "8001",
			},
			{
				Name:  "DRAGONFLY_INJECT_PROXY",
				Value: "http://$(NODE_NAME):$(DRAGONFLY_PROXY_PORT)",
			},
		},
	}
}

func (pei *ProxyEnvInjector) Inject(pod *corev1.Pod) {
	podlog.Info("ProxyEnvInjector Inject")
	// inject env to all containers
	containers := pod.Spec.Containers
	for i := range containers {
		pei.InjectContainer(&containers[i])
	}
}

func (pei *ProxyEnvInjector) InjectContainer(c *corev1.Container) {
	for _, e := range pei.Envs {
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
