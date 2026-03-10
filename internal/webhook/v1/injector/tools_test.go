package injector

import (
	"fmt"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	format.MaxLength = 0 // 0 means no limit
}

var _ = Describe("ToolsInjector", func() {
	var (
		injector             *Tools
		defaultCliToolsDir   string
		defaultMountPath     string
		defaultCliToolsImage string
		annotationImage      string
	)

	BeforeEach(func() {
		injector = NewTools()
		defaultCliToolsDir = "/usr/local/bin"
		defaultMountPath = "/d7y/bin"
		defaultCliToolsImage = "dragonflyoss/client:latest"
		annotationImage = "annotated/tools-image:v1.2.3"
	})

	// Helper function to create a clean pod object for each test
	makePod := func(name string, containers int, annotations map[string]string) *corev1.Pod {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:        name,
				Annotations: annotations,
			},
			Spec: corev1.PodSpec{},
		}
		for i := 0; i < containers; i++ {
			pod.Spec.Containers = append(pod.Spec.Containers, corev1.Container{Name: fmt.Sprintf("container-%d", i+1)})
		}
		return pod
	}

	// Helper function to create the expected volume
	makeExpectedVolume := func() []corev1.Volume {
		return []corev1.Volume{
			{
				Name:         CliToolsVolumeName,
				VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
			},
		}
	}

	// Helper function to create the expected volume mount
	makeExpectedVolumeMount := func(mountPath string) []corev1.VolumeMount {
		return []corev1.VolumeMount{
			{
				Name:      CliToolsVolumeName,
				MountPath: mountPath,
			},
			{
				Name:      CliToolsVolumeName,
				MountPath: filepath.Join(CliToolsMountDirPath, CliToolsDfgetName),
				SubPath:   CliToolsDfgetName,
				ReadOnly:  true,
			},
			{
				Name:      CliToolsVolumeName,
				MountPath: filepath.Join(CliToolsMountDirPath, CliToolsDfcacheName),
				SubPath:   CliToolsDfcacheName,
				ReadOnly:  true,
			},
			{
				Name:      CliToolsVolumeName,
				MountPath: filepath.Join(CliToolsMountDirPath, CliToolsDfstoreName),
				SubPath:   CliToolsDfstoreName,
				ReadOnly:  true,
			},
		}
	}

	// Helper function to create the expected init container
	makeExpectedInitContainer := func(image, dirPath, mountPath string) corev1.Container {
		return corev1.Container{
			Name:            CliToolsInitContainerName,
			Image:           image,
			ImagePullPolicy: corev1.PullIfNotPresent,
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      CliToolsVolumeName,
					MountPath: mountPath,
				},
			},
			Command: []string{"install",
				"-D",
				filepath.Join(dirPath, CliToolsDfgetName),
				filepath.Join(dirPath, CliToolsDfcacheName),
				filepath.Join(dirPath, CliToolsDfstoreName),
				"-t",
				defaultMountPath + "/",
			},
		}
	}

	Describe("Inject", func() {
		Context("when injecting initContainer, volume, mount", func() {
			It("should inject into a simple pod successfully", func() {
				By("creating a simple pod")
				pod := makePod("test-pod-1", 1, nil)
				config := &Config{CliToolsDirPath: defaultCliToolsDir, CliToolsImage: defaultCliToolsImage}

				By("creating expected pod")
				expectedPod := makePod("test-pod-1", 1, nil)
				expectedPod.Spec.InitContainers = []corev1.Container{
					makeExpectedInitContainer(defaultCliToolsImage, defaultCliToolsDir, defaultMountPath),
				}
				expectedPod.Spec.Volumes = makeExpectedVolume()
				expectedPod.Spec.Containers[0].VolumeMounts = makeExpectedVolumeMount(defaultMountPath)

				By("performing injection")
				injector.Inject(pod, config)

				By("verifying the result")
				Expect(pod).To(Equal(expectedPod))
			})

			It("should use image from annotation if present", func() {
				By("creating a pod with image annotation")
				pod := makePod("test-pod-2", 1, map[string]string{CliToolsImageAnnotation: annotationImage})
				config := &Config{CliToolsDirPath: defaultCliToolsDir, CliToolsImage: defaultCliToolsImage}

				By("creating expected pod with annotation image")
				expectedPod := makePod("test-pod-2", 1, map[string]string{CliToolsImageAnnotation: annotationImage})
				expectedPod.Spec.InitContainers = []corev1.Container{
					makeExpectedInitContainer(annotationImage, defaultCliToolsDir, defaultMountPath),
				}
				expectedPod.Spec.Volumes = makeExpectedVolume()
				expectedPod.Spec.Containers[0].VolumeMounts = makeExpectedVolumeMount(defaultMountPath)

				By("performing injection")
				injector.Inject(pod, config)

				By("verifying the result")
				Expect(pod).To(Equal(expectedPod))
			})

			It("should inject into multiple containers", func() {
				By("creating a pod with multiple containers")
				pod := makePod("test-pod-3", 2, nil)
				config := &Config{CliToolsDirPath: defaultCliToolsDir, CliToolsImage: defaultCliToolsImage}

				By("creating expected pod")
				expectedPod := makePod("test-pod-3", 2, nil)
				expectedPod.Spec.InitContainers = []corev1.Container{
					makeExpectedInitContainer(defaultCliToolsImage, defaultCliToolsDir, defaultMountPath),
				}
				expectedPod.Spec.Volumes = makeExpectedVolume()
				expectedPod.Spec.Containers[0].VolumeMounts = makeExpectedVolumeMount(defaultMountPath)
				expectedPod.Spec.Containers[1].VolumeMounts = makeExpectedVolumeMount(defaultMountPath)

				By("performing injection")
				injector.Inject(pod, config)

				By("verifying the result")
				Expect(pod).To(Equal(expectedPod))
			})

			It("should be idempotent and not inject if everything already exists", func() {
				By("creating a pod with everything already injected")
				pod := makePod("test-pod-4", 1, nil)
				pod.Spec.InitContainers = []corev1.Container{
					makeExpectedInitContainer(defaultCliToolsImage, defaultCliToolsDir, defaultMountPath),
				}
				pod.Spec.Volumes = makeExpectedVolume()
				pod.Spec.Containers[0].VolumeMounts = makeExpectedVolumeMount(defaultMountPath)

				By("creating expected pod (should be unchanged)")
				expectedPod := makePod("test-pod-4", 1, nil)
				expectedPod.Spec.InitContainers = []corev1.Container{
					makeExpectedInitContainer(defaultCliToolsImage, defaultCliToolsDir, defaultMountPath),
				}
				expectedPod.Spec.Volumes = makeExpectedVolume()
				expectedPod.Spec.Containers[0].VolumeMounts = makeExpectedVolumeMount(defaultMountPath)

				By("performing injection")
				config := &Config{CliToolsDirPath: defaultCliToolsDir, CliToolsImage: defaultCliToolsImage}
				injector.Inject(pod, config)

				By("verifying the result")
				Expect(pod).To(Equal(expectedPod))
			})

			It("should handle pods with no containers gracefully", func() {
				By("creating a pod with no containers")
				pod := makePod("test-pod-5", 0, nil)
				config := &Config{CliToolsDirPath: defaultCliToolsDir, CliToolsImage: defaultCliToolsImage}

				By("creating expected pod")
				expectedPod := makePod("test-pod-5", 0, nil)
				expectedPod.Spec.InitContainers = []corev1.Container{
					makeExpectedInitContainer(defaultCliToolsImage, defaultCliToolsDir, defaultMountPath),
				}
				expectedPod.Spec.Volumes = makeExpectedVolume()

				By("performing injection")
				injector.Inject(pod, config)

				By("verifying the result")
				Expect(pod).To(Equal(expectedPod))
			})

			It("should correctly inject into container-2 even if container-1 already has dependencies", func() {
				By("creating a pod where container-1 already has dependencies")
				pod := makePod("test-pod-6", 2, nil)
				pod.Spec.InitContainers = []corev1.Container{makeExpectedInitContainer(defaultCliToolsImage, defaultCliToolsDir, defaultMountPath)}
				pod.Spec.Volumes = makeExpectedVolume()
				pod.Spec.Containers[0].VolumeMounts = makeExpectedVolumeMount(defaultMountPath)

				By("creating expected pod (container-2 should also get mount and env)")
				expectedPod := makePod("test-pod-6", 2, nil)
				expectedPod.Spec.InitContainers = []corev1.Container{makeExpectedInitContainer(defaultCliToolsImage, defaultCliToolsDir, defaultMountPath)}
				expectedPod.Spec.Volumes = makeExpectedVolume()
				expectedPod.Spec.Containers[0].VolumeMounts = makeExpectedVolumeMount(defaultMountPath)
				expectedPod.Spec.Containers[1].VolumeMounts = makeExpectedVolumeMount(defaultMountPath)

				By("performing injection")
				config := &Config{CliToolsDirPath: defaultCliToolsDir, CliToolsImage: defaultCliToolsImage}
				injector.Inject(pod, config)

				By("verifying the result")
				Expect(pod).To(Equal(expectedPod))
			})
		})
	})

	Describe("CheckFunctions", func() {
		var (
			injectedPod *corev1.Pod
			emptyPod    *corev1.Pod
		)

		BeforeEach(func() {
			// Pod with everything injected
			injectedPod = &corev1.Pod{
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{{Name: CliToolsInitContainerName}},
					Volumes:        []corev1.Volume{{Name: CliToolsVolumeName}},
					Containers: []corev1.Container{
						{
							Name:         "main-container",
							VolumeMounts: []corev1.VolumeMount{{Name: CliToolsVolumeName}},
						},
					},
				},
			}

			// Empty Pod
			emptyPod = &corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "main-container"}}, // A container with no deps
				},
			}
		})

		Context("with CheckInitContainerIsExist", func() {
			It("should find existing init container", func() {
				result := injector.hasInitContainer(injectedPod)
				Expect(result).To(BeTrue())
			})

			It("should not find init container in empty pod", func() {
				result := injector.hasInitContainer(emptyPod)
				Expect(result).To(BeFalse())
			})
		})

		Context("CheckVolumeIsExist", func() {
			It("should find existing volume", func() {
				result := injector.hasVolume(injectedPod)
				Expect(result).To(BeTrue())
			})

			It("should not find volume in empty pod", func() {
				result := injector.hasVolume(emptyPod)
				Expect(result).To(BeFalse())
			})
		})

		Context("CheckVolumeMountIsExist", func() {
			var (
				containerWithMount    *corev1.Container
				containerWithoutMount *corev1.Container
			)

			BeforeEach(func() {
				containerWithMount = &injectedPod.Spec.Containers[0]
				containerWithoutMount = &emptyPod.Spec.Containers[0]
			})

			It("should find existing volume mount in container", func() {
				result := injector.hasVolumeMount(containerWithMount)
				Expect(result).To(BeTrue())
			})

			It("should not find volume mount in container", func() {
				result := injector.hasVolumeMount(containerWithoutMount)
				Expect(result).To(BeFalse())
			})
		})
	})
})
