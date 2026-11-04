/*
 *     Copyright 2026 The Dragonfly Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package v1

import (
	"context"
	"os"
	"path/filepath"

	"d7y.io/dragonfly-injector/internal/webhook/v1/injector"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// mockInjector is a mock implementation of the Injector interface for testing purposes.
// It records whether its Inject method has been called.
type mockInjector struct {
	called bool
	config *injector.Config
}

func (m *mockInjector) Inject(pod *corev1.Pod, config *injector.Config) {
	m.called = true
	m.config = config
}

func (m *mockInjector) Reset() {
	m.called = false
	m.config = nil
}

var _ = Describe("Pod Webhook", func() {
	var (
		defaulter   *PodCustomDefaulter
		mockInj     *mockInjector
		ctx         context.Context
		testPod     *corev1.Pod
		configMgr   *injector.ConfigManager
		tempDir     string
		fakeClient  client.Client
		scheme      *runtime.Scheme
		testNsName  string
		testPodName string
	)

	BeforeEach(func() {
		ctx = context.Background()
		testNsName = "test-namespace"
		testPodName = "test-pod"

		// Create a mock injector to verify the injection logic
		mockInj = &mockInjector{}

		tempDir = GinkgoT().TempDir()

		// Write a predictable config file for the test
		testConfig := &injector.Config{
			InitContainerImage: injector.InitContainerImage{
				Registry:   "docker.io",
				Repository: "dragonflyoss/client",
				Tag:        "latest",
			},
		}
		yamlData, err := yaml.Marshal(testConfig)
		Expect(err).NotTo(HaveOccurred())

		configPath := filepath.Join(tempDir, "config.yaml")
		err = os.WriteFile(configPath, yamlData, 0644)
		Expect(err).NotTo(HaveOccurred())

		// Initialize ConfigManager with the temp config path
		configMgr = injector.NewConfigManager(tempDir)

		// Initialize the scheme and fake client
		scheme = runtime.NewScheme()
		err = corev1.AddToScheme(scheme)
		Expect(err).NotTo(HaveOccurred())

		// Base test pod object
		testPod = &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:        testPodName,
				Namespace:   testNsName,
				Annotations: make(map[string]string),
			},
		}
	})

	// This function centralizes the setup of the defaulter for each context
	setupDefaulter := func(initObjs ...client.Object) {
		// Create a new fake client for each test scenario to ensure isolation
		fakeClient = fake.NewClientBuilder().WithScheme(scheme).WithObjects(initObjs...).Build()
		defaulter = NewPodCustomDefaulter(fakeClient, configMgr)
		// CRITICAL: Replace the real injectors with our mock for testing purposes
		defaulter.injectors = []Injector{mockInj}
	}

	Context("When evaluating if injection is required", func() {

		Context("and injection is enabled by Namespace label", func() {
			It("should inject the pod", func() {
				By("creating a namespace with the injection label")
				labeledNs := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: testNsName,
						Labels: map[string]string{
							injector.NamespaceInjectLabelName: injector.NamespaceInjectLabelValue,
						},
					},
				}
				setupDefaulter(labeledNs)

				By("calling the Default method")
				err := defaulter.Default(ctx, testPod)
				Expect(err).NotTo(HaveOccurred())

				By("verifying the injector was called")
				Expect(mockInj.called).To(BeTrue())
				Expect(mockInj.config).NotTo(BeNil())
			})
		})

		Context("and injection is enabled by Pod annotation", func() {
			It("should inject the pod", func() {
				By("creating a namespace without the injection label")
				unlabeledNs := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: testNsName},
				}
				setupDefaulter(unlabeledNs)

				By("annotating the pod to enable injection")
				testPod.Annotations[injector.PodInjectAnnotationName] = injector.PodInjectAnnotationValue

				By("calling the Default method")
				err := defaulter.Default(ctx, testPod)
				Expect(err).NotTo(HaveOccurred())

				By("verifying the injector was called")
				Expect(mockInj.called).To(BeTrue())
			})
		})

		Context("and injection is enabled by both Namespace and Pod", func() {
			It("should inject the pod once", func() {
				By("creating a namespace with the injection label")
				labeledNs := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: testNsName,
						Labels: map[string]string{
							injector.NamespaceInjectLabelName: injector.NamespaceInjectLabelValue,
						},
					},
				}
				setupDefaulter(labeledNs)

				By("annotating the pod to enable injection")
				testPod.Annotations[injector.PodInjectAnnotationName] = injector.PodInjectAnnotationValue

				By("calling the Default method")
				err := defaulter.Default(ctx, testPod)
				Expect(err).NotTo(HaveOccurred())

				By("verifying the injector was called")
				Expect(mockInj.called).To(BeTrue())
			})
		})

		Context("and injection is not required", func() {
			It("should not inject the pod unless label or annotation is present", func() {
				By("creating a namespace without the injection label")
				unlabeledNs := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNsName}}
				setupDefaulter(unlabeledNs)

				By("calling the Default method on a pod with no relevant annotation")
				err := defaulter.Default(ctx, testPod)
				Expect(err).NotTo(HaveOccurred())

				By("verifying the injector was NOT called")
				Expect(mockInj.called).To(BeFalse())
			})

			It("should not inject if the namespace label has the wrong value", func() {
				By("creating a namespace with an incorrect injection label value")
				incorrectlyLabeledNs := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name:   testNsName,
						Labels: map[string]string{injector.NamespaceInjectLabelName: "disabled"},
					},
				}
				setupDefaulter(incorrectlyLabeledNs)

				By("calling the Default method")
				err := defaulter.Default(ctx, testPod)
				Expect(err).NotTo(HaveOccurred())

				By("verifying the injector was NOT called")
				Expect(mockInj.called).To(BeFalse())
			})

			It("should not inject if the pod annotation has the wrong value", func() {
				By("creating a namespace without the injection label")
				unlabeledNs := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNsName}}
				setupDefaulter(unlabeledNs)

				By("annotating the pod with an incorrect value")
				testPod.Annotations[injector.PodInjectAnnotationName] = "false"

				By("calling the Default method")
				err := defaulter.Default(ctx, testPod)
				Expect(err).NotTo(HaveOccurred())

				By("verifying the injector was NOT called")
				Expect(mockInj.called).To(BeFalse())
			})

			It("should not inject if the pod's namespace cannot be found", func() {
				By("creating a fake client with NO namespace object")
				setupDefaulter() // No objects added to client

				By("calling the Default method")
				err := defaulter.Default(ctx, testPod)
				Expect(err).NotTo(HaveOccurred()) // The method logs error but returns nil

				By("verifying the injector was NOT called")
				Expect(mockInj.called).To(BeFalse())
			})
		})

		Context("when the object is not a Pod", func() {
			It("should return an error", func() {
				By("creating a non-pod object")
				notAPod := &corev1.ConfigMap{}
				setupDefaulter()

				By("calling the Default method")
				err := defaulter.Default(ctx, notAPod)

				By("verifying that an error is returned")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("expected an Pod object but got"))
				Expect(mockInj.called).To(BeFalse())
			})
		})
	})

	// The contexts below use the real injectors instead of the mock so that the
	// actual fields added to the pod spec are asserted.
	Context("When mutating a pod with the real injectors", func() {
		var (
			realDefaulter *PodCustomDefaulter
			injectionPod  *corev1.Pod
			configDir     string
		)

		BeforeEach(func() {
			configDir = GinkgoT().TempDir()
			writeTestInjectorConfig(configDir)

			unlabeledNs := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNsName}}
			fakeClient = fake.NewClientBuilder().WithScheme(scheme).WithObjects(unlabeledNs).Build()
			realDefaulter = NewPodCustomDefaulter(fakeClient, injector.NewConfigManager(configDir))

			injectionPod = newInjectionPod(map[string]string{
				injector.PodInjectAnnotationName: injector.PodInjectAnnotationValue,
			})
		})

		It("injects the dfdaemon socket volume, the binaries volume and the init container", func() {
			Expect(realDefaulter.Default(ctx, injectionPod)).To(Succeed())

			By("checking the dfdaemon unix socket volume")
			socketVolume := findPodVolume(injectionPod, injector.DfdaemonUnixSockVolumeName)
			Expect(socketVolume).NotTo(BeNil(), "the dfdaemon socket volume should be injected")
			Expect(socketVolume.VolumeSource.HostPath).NotTo(BeNil())
			Expect(socketVolume.VolumeSource.HostPath.Path).To(Equal(injector.DfdaemonUnixSockPath))
			Expect(socketVolume.VolumeSource.HostPath.Type).NotTo(BeNil())
			Expect(*socketVolume.VolumeSource.HostPath.Type).To(Equal(corev1.HostPathSocket))

			By("checking the binaries volume")
			binariesVolume := findPodVolume(injectionPod, injector.BinaryVolumeName)
			Expect(binariesVolume).NotTo(BeNil(), "the binaries volume should be injected")
			Expect(binariesVolume.VolumeSource.EmptyDir).NotTo(BeNil())

			By("checking the binaries init container")
			initContainer := findPodContainer(injectionPod.Spec.InitContainers, injector.InitContainerName)
			Expect(initContainer).NotTo(BeNil(), "the binaries init container should be injected")
			Expect(initContainer.Image).To(Equal("registry.example.com/dragonflyoss/client:test-tag"))
			Expect(initContainer.ImagePullPolicy).To(Equal(corev1.PullAlways))
			Expect(initContainer.Command).To(ContainElement(filepath.Join(injector.BinaryDirPath, injector.DfgetBinaryName)))
			Expect(initContainer.VolumeMounts).To(HaveLen(1))
			Expect(initContainer.VolumeMounts[0].Name).To(Equal(injector.BinaryVolumeName))
			Expect(initContainer.VolumeMounts[0].MountPath).To(Equal(injector.BinaryVolumeMountPath))

			By("checking the image pull secrets coming from the config")
			Expect(injectionPod.Spec.ImagePullSecrets).To(ConsistOf(corev1.LocalObjectReference{Name: "dragonfly-pull-secret"}))

			By("checking that the application container receives all the mounts")
			appContainer := findPodContainer(injectionPod.Spec.Containers, "app")
			Expect(appContainer).NotTo(BeNil())
			Expect(appContainer.VolumeMounts).To(HaveLen(7))
			Expect(volumeMountPaths(appContainer)).To(ContainElements(
				injector.DfdaemonUnixSockPath,
				injector.BinaryVolumeMountPath,
				filepath.Join(injector.BinaryMountDirPath, injector.DfgetBinaryName),
				filepath.Join(injector.BinaryMountDirPath, injector.DfdaemonBinaryName),
			))
		})

		It("honours the init container image override annotation", func() {
			injectionPod.Annotations[injector.InitContainerImageAnnotation] = "example.com/custom/client:custom"

			Expect(realDefaulter.Default(ctx, injectionPod)).To(Succeed())

			initContainer := findPodContainer(injectionPod.Spec.InitContainers, injector.InitContainerName)
			Expect(initContainer).NotTo(BeNil())
			Expect(initContainer.Image).To(Equal("example.com/custom/client:custom"))
		})

		It("is idempotent when the same pod is defaulted twice", func() {
			Expect(realDefaulter.Default(ctx, injectionPod)).To(Succeed())
			firstPass := injectionPod.DeepCopy()

			Expect(realDefaulter.Default(ctx, injectionPod)).To(Succeed())

			Expect(injectionPod).To(Equal(firstPass))
			Expect(injectionPod.Spec.InitContainers).To(HaveLen(1))
			Expect(injectionPod.Spec.Volumes).To(HaveLen(2))
			Expect(findPodContainer(injectionPod.Spec.Containers, "app").VolumeMounts).To(HaveLen(7))
		})

		It("skips only the unix socket injection when the skip annotation is set", func() {
			injectionPod.Annotations[injector.SkipUnixSockInjectAnnotationName] = injector.SkipUnixSockInjectAnnotationValue

			Expect(realDefaulter.Default(ctx, injectionPod)).To(Succeed())

			Expect(findPodVolume(injectionPod, injector.DfdaemonUnixSockVolumeName)).To(BeNil())
			Expect(findPodVolume(injectionPod, injector.BinaryVolumeName)).NotTo(BeNil())

			appContainer := findPodContainer(injectionPod.Spec.Containers, "app")
			Expect(appContainer).NotTo(BeNil())
			Expect(appContainer.VolumeMounts).To(HaveLen(6))
			Expect(volumeMountPaths(appContainer)).NotTo(ContainElement(injector.DfdaemonUnixSockPath))
		})

		DescribeTable("mutates the pod when injection is requested",
			func(namespaceLabels, podAnnotations map[string]string) {
				ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNsName, Labels: namespaceLabels}}
				c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(ns).Build()
				d := NewPodCustomDefaulter(c, injector.NewConfigManager(configDir))

				pod := newInjectionPod(podAnnotations)
				Expect(d.Default(ctx, pod)).To(Succeed())

				Expect(findPodVolume(pod, injector.DfdaemonUnixSockVolumeName)).NotTo(BeNil())
				Expect(findPodContainer(pod.Spec.InitContainers, injector.InitContainerName)).NotTo(BeNil())
			},
			Entry("pod annotation is true", nil,
				map[string]string{injector.PodInjectAnnotationName: injector.PodInjectAnnotationValue}),
			Entry("namespace label is true",
				map[string]string{injector.NamespaceInjectLabelName: injector.NamespaceInjectLabelValue}, nil),
			Entry("namespace label and pod annotation are true",
				map[string]string{injector.NamespaceInjectLabelName: injector.NamespaceInjectLabelValue},
				map[string]string{injector.PodInjectAnnotationName: injector.PodInjectAnnotationValue}),
		)

		DescribeTable("leaves the pod untouched when injection is not requested",
			func(namespaceLabels, podAnnotations map[string]string) {
				ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNsName, Labels: namespaceLabels}}
				c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(ns).Build()
				d := NewPodCustomDefaulter(c, injector.NewConfigManager(configDir))

				pod := newInjectionPod(podAnnotations)
				before := pod.DeepCopy()

				Expect(d.Default(ctx, pod)).To(Succeed())

				Expect(pod).To(Equal(before))
			},
			Entry("neither a namespace label nor a pod annotation is present", nil, nil),
			Entry("pod annotation is not true", nil,
				map[string]string{injector.PodInjectAnnotationName: "false"}),
			Entry("namespace label is not true",
				map[string]string{injector.NamespaceInjectLabelName: "disabled"}, nil),
		)

		Context("and the AdmissionReview payload is malformed", func() {
			It("decodes a valid review and mutates the decoded pod", func() {
				request, pod, err := decodeAdmissionReview(admissionReviewForPod(true))
				Expect(err).NotTo(HaveOccurred())
				Expect(request).NotTo(BeNil())
				Expect(request.Name).To(Equal("test-pod"))
				Expect(pod).NotTo(BeNil())
				Expect(pod.Name).To(Equal("test-pod"))

				Expect(realDefaulter.Default(ctx, pod)).To(Succeed())
				Expect(findPodVolume(pod, injector.DfdaemonUnixSockVolumeName)).NotTo(BeNil())
				Expect(findPodContainer(pod.Spec.InitContainers, injector.InitContainerName)).NotTo(BeNil())
			})

			DescribeTable("returns an explicit error instead of panicking",
				func(payload []byte, wantError string) {
					_, pod, err := decodeAdmissionReview(payload)

					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring(wantError))
					Expect(pod).To(BeNil())
				},
				Entry("the body is not valid JSON", []byte(`{"kind":`),
					"failed to decode admission review"),
				Entry("the request is null",
					[]byte(`{"apiVersion":"admission.k8s.io/v1","kind":"AdmissionReview","request":null}`),
					"does not contain a request"),
				Entry("the object raw field is empty",
					[]byte(`{"apiVersion":"admission.k8s.io/v1","kind":"AdmissionReview","request":{"uid":"empty","object":null}}`),
					"no content to decode"),
				Entry("the object is not a pod",
					[]byte(`{"apiVersion":"admission.k8s.io/v1","kind":"AdmissionReview","request":{"uid":"configmap","object":{"apiVersion":"v1","kind":"ConfigMap","metadata":{"name":"config"}}}}`),
					"unable to decode"),
			)
		})
	})
})

// writeTestInjectorConfig writes a predictable injector.yaml to dir so that the
// assertions can rely on a known init container image and pull secret.
func writeTestInjectorConfig(dir string) {
	config := &injector.Config{
		InitContainerImage: injector.InitContainerImage{
			Registry:    "registry.example.com",
			Repository:  "dragonflyoss/client",
			Tag:         "test-tag",
			PullPolicy:  corev1.PullAlways,
			PullSecrets: []corev1.LocalObjectReference{{Name: "dragonfly-pull-secret"}},
		},
	}

	data, err := yaml.Marshal(config)
	Expect(err).NotTo(HaveOccurred())
	Expect(os.WriteFile(filepath.Join(dir, "injector.yaml"), data, 0o644)).To(Succeed())
}

// newInjectionPod returns a minimal pod that can be used to exercise the
// webhook mutation logic.
func newInjectionPod(annotations map[string]string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-pod",
			Namespace:   "test-namespace",
			Annotations: annotations,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "app", Image: "busybox:1.36"}},
		},
	}
}

// findPodVolume returns the volume with the given name, or nil when it is not
// present.
func findPodVolume(pod *corev1.Pod, name string) *corev1.Volume {
	for i := range pod.Spec.Volumes {
		if pod.Spec.Volumes[i].Name == name {
			return &pod.Spec.Volumes[i]
		}
	}

	return nil
}

// findPodContainer returns the container with the given name, or nil when it is
// not present.
func findPodContainer(containers []corev1.Container, name string) *corev1.Container {
	for i := range containers {
		if containers[i].Name == name {
			return &containers[i]
		}
	}

	return nil
}

// volumeMountPaths returns the mount paths of the given container.
func volumeMountPaths(container *corev1.Container) []string {
	paths := make([]string, 0, len(container.VolumeMounts))
	for _, mount := range container.VolumeMounts {
		paths = append(paths, mount.MountPath)
	}

	return paths
}
