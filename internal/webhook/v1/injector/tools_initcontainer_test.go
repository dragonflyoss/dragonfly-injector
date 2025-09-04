package injector

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestToolsInitcontainerInjector_Inject(t *testing.T) {
	injector := NewToolsInitcontainerInjector()

	// Define common variables for test cases to keep them DRY
	defaultCliToolsDir := "/opt/df-tools"
	defaultMountPath := filepath.Clean(defaultCliToolsDir) + "-mount"
	defaultCliToolsImage := "default/tools-image:latest"
	annotationImage := "annotated/tools-image:v1.2.3"

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
	makeExpectedVolume := func() corev1.Volume {
		return corev1.Volume{
			Name:         CliToolsVolumeName,
			VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
		}
	}

	// Helper function to create the expected volume mount
	makeExpectedVolumeMount := func(mountPath string) corev1.VolumeMount {
		return corev1.VolumeMount{
			Name:      CliToolsVolumeName,
			MountPath: mountPath,
		}
	}

	// Helper function to create the expected env var
	makeExpectedEnvVar := func(mountPath string) corev1.EnvVar {
		return corev1.EnvVar{
			Name:  CliToolsPathEnvName,
			Value: mountPath,
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
			Command: []string{"cp", "-rf", dirPath + "/.", mountPath + "/"},
		}
	}

	testCases := []struct {
		name        string
		config      *InjectConf
		pod         *corev1.Pod
		expectedPod *corev1.Pod
	}{
		{
			name:   "should inject initContainer, volume, mount, and env into a simple pod",
			config: &InjectConf{CliToolsDirPath: defaultCliToolsDir, CliToolsImage: defaultCliToolsImage},
			pod:    makePod("test-pod-1", 1, nil),
			expectedPod: func() *corev1.Pod {
				p := makePod("test-pod-1", 1, nil)
				p.Spec.InitContainers = []corev1.Container{
					makeExpectedInitContainer(defaultCliToolsImage, defaultCliToolsDir, defaultMountPath),
				}
				p.Spec.Volumes = []corev1.Volume{makeExpectedVolume()}
				p.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{makeExpectedVolumeMount(defaultMountPath)}
				p.Spec.Containers[0].Env = []corev1.EnvVar{makeExpectedEnvVar(defaultMountPath)}
				return p
			}(),
		},
		{
			name:   "should use image from annotation if present",
			config: &InjectConf{CliToolsDirPath: defaultCliToolsDir, CliToolsImage: defaultCliToolsImage},
			pod:    makePod("test-pod-2", 1, map[string]string{CliToolsImageAnnotation: annotationImage}),
			expectedPod: func() *corev1.Pod {
				p := makePod("test-pod-2", 1, map[string]string{CliToolsImageAnnotation: annotationImage})
				// The image should come from the annotation
				p.Spec.InitContainers = []corev1.Container{
					makeExpectedInitContainer(annotationImage, defaultCliToolsDir, defaultMountPath),
				}
				p.Spec.Volumes = []corev1.Volume{makeExpectedVolume()}
				p.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{makeExpectedVolumeMount(defaultMountPath)}
				p.Spec.Containers[0].Env = []corev1.EnvVar{makeExpectedEnvVar(defaultMountPath)}
				return p
			}(),
		},
		{
			name:   "should inject into multiple containers",
			config: &InjectConf{CliToolsDirPath: defaultCliToolsDir, CliToolsImage: defaultCliToolsImage},
			pod:    makePod("test-pod-3", 2, nil),
			expectedPod: func() *corev1.Pod {
				p := makePod("test-pod-3", 2, nil)
				p.Spec.InitContainers = []corev1.Container{
					makeExpectedInitContainer(defaultCliToolsImage, defaultCliToolsDir, defaultMountPath),
				}
				p.Spec.Volumes = []corev1.Volume{makeExpectedVolume()}
				// Both containers should get the mount and env
				p.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{makeExpectedVolumeMount(defaultMountPath)}
				p.Spec.Containers[0].Env = []corev1.EnvVar{makeExpectedEnvVar(defaultMountPath)}
				p.Spec.Containers[1].VolumeMounts = []corev1.VolumeMount{makeExpectedVolumeMount(defaultMountPath)}
				p.Spec.Containers[1].Env = []corev1.EnvVar{makeExpectedEnvVar(defaultMountPath)}
				return p
			}(),
		},
		{
			name:   "should be idempotent and not inject if everything already exists",
			config: &InjectConf{CliToolsDirPath: defaultCliToolsDir, CliToolsImage: defaultCliToolsImage},
			pod: func() *corev1.Pod {
				p := makePod("test-pod-4", 1, nil)
				p.Spec.InitContainers = []corev1.Container{
					makeExpectedInitContainer(defaultCliToolsImage, defaultCliToolsDir, defaultMountPath),
				}
				p.Spec.Volumes = []corev1.Volume{makeExpectedVolume()}
				p.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{makeExpectedVolumeMount(defaultMountPath)}
				p.Spec.Containers[0].Env = []corev1.EnvVar{makeExpectedEnvVar(defaultMountPath)}
				return p
			}(),
			expectedPod: func() *corev1.Pod {
				p := makePod("test-pod-4", 1, nil)
				p.Spec.InitContainers = []corev1.Container{
					makeExpectedInitContainer(defaultCliToolsImage, defaultCliToolsDir, defaultMountPath),
				}
				p.Spec.Volumes = []corev1.Volume{makeExpectedVolume()}
				p.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{makeExpectedVolumeMount(defaultMountPath)}
				p.Spec.Containers[0].Env = []corev1.EnvVar{makeExpectedEnvVar(defaultMountPath)}
				return p
			}(),
		},
		{
			name:   "should handle pods with no containers gracefully",
			config: &InjectConf{CliToolsDirPath: defaultCliToolsDir, CliToolsImage: defaultCliToolsImage},
			pod:    makePod("test-pod-5", 0, nil),
			expectedPod: func() *corev1.Pod {
				p := makePod("test-pod-5", 0, nil)
				p.Spec.InitContainers = []corev1.Container{
					makeExpectedInitContainer(defaultCliToolsImage, defaultCliToolsDir, defaultMountPath),
				}
				p.Spec.Volumes = []corev1.Volume{makeExpectedVolume()}
				// Spec.Containers remains empty
				return p
			}(),
		},
		{
			// This test now confirms the bug fix in `CheckVolumeMountIsExist`
			name:   "should correctly inject into container-2 even if container-1 already has dependencies",
			config: &InjectConf{CliToolsDirPath: defaultCliToolsDir, CliToolsImage: defaultCliToolsImage},
			pod: func() *corev1.Pod {
				p := makePod("test-pod-6", 2, nil)
				// The initContainer and Volume already exist
				p.Spec.InitContainers = []corev1.Container{makeExpectedInitContainer(defaultCliToolsImage, defaultCliToolsDir, defaultMountPath)}
				p.Spec.Volumes = []corev1.Volume{makeExpectedVolume()}
				// container-1 already has the mount and env, but container-2 does not
				p.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{makeExpectedVolumeMount(defaultMountPath)}
				p.Spec.Containers[0].Env = []corev1.EnvVar{makeExpectedEnvVar(defaultMountPath)}
				return p
			}(),
			expectedPod: func() *corev1.Pod {
				p := makePod("test-pod-6", 2, nil)
				p.Spec.InitContainers = []corev1.Container{makeExpectedInitContainer(defaultCliToolsImage, defaultCliToolsDir, defaultMountPath)}
				p.Spec.Volumes = []corev1.Volume{makeExpectedVolume()}
				// The correct behavior is that container-2 also gets the mount and env
				p.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{makeExpectedVolumeMount(defaultMountPath)}
				p.Spec.Containers[0].Env = []corev1.EnvVar{makeExpectedEnvVar(defaultMountPath)}
				p.Spec.Containers[1].VolumeMounts = []corev1.VolumeMount{makeExpectedVolumeMount(defaultMountPath)}
				p.Spec.Containers[1].Env = []corev1.EnvVar{makeExpectedEnvVar(defaultMountPath)}
				return p
			}(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			injector.Inject(tc.pod, tc.config)
			assert.Equal(t, tc.expectedPod, tc.pod, "The pod after injection should match the expected pod.")
		})
	}
}

// It's good practice to test helper functions individually.
func TestToolsInitcontainerInjector_CheckFunctions(t *testing.T) {
	injector := NewToolsInitcontainerInjector()

	// Pod with everything injected
	injectedPod := &corev1.Pod{
		Spec: corev1.PodSpec{
			InitContainers: []corev1.Container{{Name: CliToolsInitContainerName}},
			Volumes:        []corev1.Volume{{Name: CliToolsVolumeName}},
			Containers: []corev1.Container{
				{
					Name:         "main-container",
					VolumeMounts: []corev1.VolumeMount{{Name: CliToolsVolumeName}},
					Env:          []corev1.EnvVar{{Name: CliToolsPathEnvName}},
				},
			},
		},
	}

	// Empty Pod
	emptyPod := &corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "main-container"}}, // A container with no deps
		},
	}

	// --- Test CheckInitContainerIsExist ---
	assert.True(t, injector.CheckInitContainerIsExist(injectedPod), "Should find existing init container")
	assert.False(t, injector.CheckInitContainerIsExist(emptyPod), "Should not find init container in empty pod")

	// --- Test CheckVolumeIsExist ---
	assert.True(t, injector.CheckVolumeIsExist(injectedPod), "Should find existing volume")
	assert.False(t, injector.CheckVolumeIsExist(emptyPod), "Should not find volume in empty pod")

	// --- Test CheckEnvIsExist ---
	containerWithEnv := &injectedPod.Spec.Containers[0]
	containerWithoutEnv := &emptyPod.Spec.Containers[0]
	assert.True(t, injector.CheckEnvIsExist(containerWithEnv), "Should find existing env var")
	assert.False(t, injector.CheckEnvIsExist(containerWithoutEnv), "Should not find env var in empty container")
	assert.False(t, injector.CheckEnvIsExist(nil), "Should handle nil container gracefully for env check")

	// --- Test CheckVolumeMountIsExist (FIXED) ---
	containerWithMount := &injectedPod.Spec.Containers[0]
	containerWithoutMount := &emptyPod.Spec.Containers[0]
	assert.True(t, injector.CheckVolumeMountIsExist(containerWithMount), "Should find existing volume mount in container")
	assert.False(t, injector.CheckVolumeMountIsExist(containerWithoutMount), "Should not find volume mount in container")
	assert.False(t, injector.CheckVolumeMountIsExist(nil), "Should handle nil container gracefully for volume mount check")
}
