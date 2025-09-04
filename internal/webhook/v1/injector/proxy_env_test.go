/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package injector

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("ProxyEnvInjector", func() {
	var (
		injector *ProxyEnvInjector
	)

	BeforeEach(func() {
		injector = NewProxyEnvInjector()
	})

	Context("when injecting proxy environment variables", func() {
		It("should inject into a pod with a single container", func() {
			By("creating a test pod with one container")
			config := &InjectConf{ProxyPort: 8888}
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "test-pod-1"},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "container-1"},
					},
				},
			}

			By("performing injection")
			injector.Inject(pod, config)

			By("verifying the injected environment variables")
			Expect(pod.Spec.Containers).To(HaveLen(1))
			container := pod.Spec.Containers[0]
			Expect(container.Env).To(ContainElements(
				corev1.EnvVar{
					Name: NodeNameEnvName,
					ValueFrom: &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{FieldPath: "spec.nodeName"},
					},
				},
				corev1.EnvVar{
					Name:  ProxyPortEnvName,
					Value: "8888",
				},
				corev1.EnvVar{
					Name:  ProxyEnvName,
					Value: fmt.Sprintf("http://$(%s):$(%s)", NodeNameEnvName, ProxyPortEnvName),
				},
			))
		})

		It("should skip injection if proxy env var already exists", func() {
			By("creating a pod with existing proxy environment variable")
			config := &InjectConf{ProxyPort: 8888}
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "test-pod-2"},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "container-1",
							Env: []corev1.EnvVar{
								{
									Name:  ProxyEnvName,
									Value: "http://my-custom-proxy:8080",
								},
							},
						},
					},
				},
			}

			By("performing injection")
			injector.Inject(pod, config)

			By("verifying the original value is preserved")
			Expect(pod.Spec.Containers).To(HaveLen(1))
			container := pod.Spec.Containers[0]
			Expect(container.Env).To(ContainElement(
				corev1.EnvVar{
					Name:  ProxyEnvName,
					Value: "http://my-custom-proxy:8080",
				},
			))
			Expect(container.Env).To(ContainElements(
				corev1.EnvVar{
					Name: NodeNameEnvName,
					ValueFrom: &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{FieldPath: "spec.nodeName"},
					},
				},
				corev1.EnvVar{
					Name:  ProxyPortEnvName,
					Value: "8888",
				},
			))
		})

		It("should inject into multiple containers", func() {
			By("creating a pod with multiple containers")
			config := &InjectConf{ProxyPort: 9999}
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "test-pod-3"},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "app-container"},
						{
							Name: "sidecar-container",
							Env: []corev1.EnvVar{
								{Name: "EXISTING_VAR", Value: "some-value"},
							},
						},
					},
				},
			}

			By("performing injection")
			injector.Inject(pod, config)

			By("verifying all containers have proxy environment variables")
			Expect(pod.Spec.Containers).To(HaveLen(2))

			for _, container := range pod.Spec.Containers {
				Expect(container.Env).To(ContainElements(
					corev1.EnvVar{
						Name: NodeNameEnvName,
						ValueFrom: &corev1.EnvVarSource{
							FieldRef: &corev1.ObjectFieldSelector{FieldPath: "spec.nodeName"},
						},
					},
					corev1.EnvVar{
						Name:  ProxyPortEnvName,
						Value: "9999",
					},
					corev1.EnvVar{
						Name:  ProxyEnvName,
						Value: fmt.Sprintf("http://$(%s):$(%s)", NodeNameEnvName, ProxyPortEnvName),
					},
				))
			}
		})

		It("should do nothing for a pod with no containers", func() {
			By("creating a pod with no containers")
			config := &InjectConf{ProxyPort: 8888}
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "test-pod-4"},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{},
				},
			}

			By("performing injection")
			injector.Inject(pod, config)

			By("verifying no containers were added")
			Expect(pod.Spec.Containers).To(BeEmpty())
		})

		It("should skip all proxy env vars if they already exist", func() {
			By("creating a pod with all proxy environment variables already existing")
			config := &InjectConf{ProxyPort: 8888}
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "test-pod-5"},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "container-1",
							Env: []corev1.EnvVar{
								{Name: NodeNameEnvName, Value: "node-1"},
								{Name: ProxyPortEnvName, Value: "1234"},
								{Name: ProxyEnvName, Value: "http://existing.proxy:1234"},
							},
						},
					},
				},
			}

			By("performing injection")
			injector.Inject(pod, config)

			By("verifying the pod remains completely unchanged")
			Expect(pod.Spec.Containers).To(HaveLen(1))
			container := pod.Spec.Containers[0]
			Expect(container.Env).To(ConsistOf(
				corev1.EnvVar{Name: NodeNameEnvName, Value: "node-1"},
				corev1.EnvVar{Name: ProxyPortEnvName, Value: "1234"},
				corev1.EnvVar{Name: ProxyEnvName, Value: "http://existing.proxy:1234"},
			))
		})
	})

	Context("when generating environment variables from configuration", func() {
		It("should return the correct environment variables", func() {
			By("creating a configuration with port 8080")
			config := &InjectConf{ProxyPort: 8080}

			By("generating environment variables")
			envs := envsFromConfig(config)

			By("verifying the generated environment variables")
			Expect(envs).To(Equal([]corev1.EnvVar{
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
					Value: "8080",
				},
				{
					Name:  ProxyEnvName,
					Value: "http://$(" + NodeNameEnvName + "):$(" + ProxyPortEnvName + ")",
				},
			}))
		})
	})
})
