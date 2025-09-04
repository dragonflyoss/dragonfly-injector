package injector

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Helper function to create the expected Volume, avoiding repetitive definitions in test cases
func makeExpectedVolume() corev1.Volume {
	hostPathType := corev1.HostPathSocket
	return corev1.Volume{
		Name: DfdaemonUnixSockVolumeName,
		VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: DfdaemonUnixSockPath,
				Type: &hostPathType,
			},
		},
	}
}

// Helper function to create the expected VolumeMount, avoiding repetitive definitions
func makeExpectedVolumeMount() corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      DfdaemonUnixSockVolumeName,
		MountPath: DfdaemonUnixSockPath,
	}
}

func TestUnixSocketInjector_Inject(t *testing.T) {
	injector := NewUnixSocketInjector()
	expectedVolume := makeExpectedVolume()
	expectedVolumeMount := makeExpectedVolumeMount()

	testCases := []struct {
		name        string
		pod         *corev1.Pod
		expectedPod *corev1.Pod
	}{
		{
			name: "Inject into a pod with no existing volume or volume mounts",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "test-pod-1"},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "container-1"}},
				},
			},
			expectedPod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "test-pod-1"},
				Spec: corev1.PodSpec{
					Volumes:    []corev1.Volume{expectedVolume},
					Containers: []corev1.Container{{Name: "container-1", VolumeMounts: []corev1.VolumeMount{expectedVolumeMount}}},
				},
			},
		},
		{
			name: "Inject into a pod where the volume already exists",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "test-pod-2"},
				Spec: corev1.PodSpec{
					Volumes:    []corev1.Volume{expectedVolume}, // Volume already exists
					Containers: []corev1.Container{{Name: "container-1"}},
				},
			},
			expectedPod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "test-pod-2"},
				Spec: corev1.PodSpec{
					Volumes:    []corev1.Volume{expectedVolume}, // Volume remains unchanged
					Containers: []corev1.Container{{Name: "container-1", VolumeMounts: []corev1.VolumeMount{expectedVolumeMount}}},
				},
			},
		},
		{
			name: "Inject into a pod where one container already has the volume mount",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "test-pod-3"},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "container-a"}, // Needs injection
						{
							Name:         "container-b", // Already exists, should not be injected again
							VolumeMounts: []corev1.VolumeMount{expectedVolumeMount},
						},
					},
				},
			},
			expectedPod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "test-pod-3"},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{expectedVolume}, // Volume will be injected
					Containers: []corev1.Container{
						{
							Name:         "container-a",
							VolumeMounts: []corev1.VolumeMount{expectedVolumeMount},
						},
						{
							Name:         "container-b", // Remains unchanged
							VolumeMounts: []corev1.VolumeMount{expectedVolumeMount},
						},
					},
				},
			},
		},
		{
			name: "Do nothing if both volume and volume mounts already exist (idempotency)",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "test-pod-4"},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{expectedVolume},
					Containers: []corev1.Container{
						{
							Name:         "container-1",
							VolumeMounts: []corev1.VolumeMount{expectedVolumeMount},
						},
					},
				},
			},
			expectedPod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "test-pod-4"},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{expectedVolume},
					Containers: []corev1.Container{
						{
							Name:         "container-1",
							VolumeMounts: []corev1.VolumeMount{expectedVolumeMount},
						},
					},
				},
			},
		},
		{
			name: "Handle a pod with no containers gracefully",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "test-pod-5"},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{}, // No containers
				},
			},
			expectedPod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "test-pod-5"},
				Spec: corev1.PodSpec{
					Volumes:    []corev1.Volume{expectedVolume}, // Volume is still injected
					Containers: []corev1.Container{},            // Container list remains empty
				},
			},
		},
		{
			name: "Inject into a pod with existing unrelated volumes and mounts",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "test-pod-6"},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{{Name: "other-volume", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}}},
					Containers: []corev1.Container{
						{
							Name:         "container-1",
							VolumeMounts: []corev1.VolumeMount{{Name: "other-mount", MountPath: "/data"}},
						},
					},
				},
			},
			expectedPod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "test-pod-6"},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{Name: "other-volume", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
						expectedVolume, // New volume is appended
					},
					Containers: []corev1.Container{
						{
							Name: "container-1",
							VolumeMounts: []corev1.VolumeMount{
								{Name: "other-mount", MountPath: "/data"},
								expectedVolumeMount, // New volume mount is appended
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Note: InjectConf is not used by this injector, so pass an empty instance
			injector.Inject(tc.pod, &InjectConf{})

			// Use testify/assert for deep comparison
			assert.Equal(t, tc.expectedPod, tc.pod, "The pod after injection should match the expected pod")
		})
	}
}
