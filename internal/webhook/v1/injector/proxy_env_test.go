package injector

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestProxyEnvInjector_Inject(t *testing.T) {
	// Create a ProxyEnvInjector instance
	injector := NewProxyEnvInjector()

	// Define test cases
	testCases := []struct {
		name        string
		config      *InjectConf
		pod         *corev1.Pod
		expectedPod *corev1.Pod
	}{
		{
			name:   "Inject into a pod with a single container",
			config: &InjectConf{ProxyPort: 8888},
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "test-pod-1"},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "container-1",
						},
					},
				},
			},
			expectedPod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "test-pod-1"},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "container-1",
							Env: []corev1.EnvVar{
								{
									Name: NodeNameEnvName,
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{FieldPath: "spec.nodeName"},
									},
								},
								{
									Name:  ProxyPortEnvName,
									Value: "8888",
								},
								{
									Name:  ProxyEnvName,
									Value: fmt.Sprintf("http://$(%s):$(%s)", NodeNameEnvName, ProxyPortEnvName),
								},
							},
						},
					},
				},
			},
		},
		{
			name:   "Skip injection if env var already exists",
			config: &InjectConf{ProxyPort: 8888},
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "test-pod-2"},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "container-1",
							Env: []corev1.EnvVar{
								{
									Name:  ProxyEnvName, // This environment variable already exists
									Value: "http://my-custom-proxy:8080",
								},
							},
						},
					},
				},
			},
			expectedPod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "test-pod-2"},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "container-1",
							Env: []corev1.EnvVar{
								{
									Name:  ProxyEnvName, // Original value should be preserved
									Value: "http://my-custom-proxy:8080",
								},
								{
									Name: NodeNameEnvName,
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{FieldPath: "spec.nodeName"},
									},
								},
								{
									Name:  ProxyPortEnvName,
									Value: "8888",
								},
							},
						},
					},
				},
			},
		},
		{
			name:   "Inject into multiple containers",
			config: &InjectConf{ProxyPort: 9999},
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "test-pod-3"},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "app-container", // No environment variables
						},
						{
							Name: "sidecar-container", // Already has one environment variable
							Env: []corev1.EnvVar{
								{Name: "EXISTING_VAR", Value: "some-value"},
							},
						},
					},
				},
			},
			expectedPod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "test-pod-3"},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "app-container",
							Env: []corev1.EnvVar{
								{
									Name: NodeNameEnvName,
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{FieldPath: "spec.nodeName"},
									},
								},
								{Name: ProxyPortEnvName, Value: "9999"},
								{Name: ProxyEnvName, Value: fmt.Sprintf("http://$(%s):$(%s)", NodeNameEnvName, ProxyPortEnvName)},
							},
						},
						{
							Name: "sidecar-container",
							Env: []corev1.EnvVar{
								{Name: "EXISTING_VAR", Value: "some-value"},
								{
									Name: NodeNameEnvName,
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{FieldPath: "spec.nodeName"},
									},
								},
								{Name: ProxyPortEnvName, Value: "9999"},
								{Name: ProxyEnvName, Value: fmt.Sprintf("http://$(%s):$(%s)", NodeNameEnvName, ProxyPortEnvName)},
							},
						},
					},
				},
			},
		},
		{
			name:   "Do nothing for a pod with no containers",
			config: &InjectConf{ProxyPort: 8888},
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "test-pod-4"},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{}, // No containers
				},
			},
			expectedPod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "test-pod-4"},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{}, // Still no containers
				},
			},
		},
		{
			name:   "All proxy env vars already exist",
			config: &InjectConf{ProxyPort: 8888},
			pod: &corev1.Pod{
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
			},
			expectedPod: &corev1.Pod{ // Pod should remain completely unchanged
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
			},
		},
	}

	// Execute all test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Perform injection
			injector.Inject(tc.pod, tc.config)

			// Use testify/assert for deep comparison, which provides clearer diff information
			assert.Equal(t, tc.expectedPod, tc.pod, "The pod after injection should match the expected pod")
		})
	}
}

func Test_envsFromConfig(t *testing.T) {
	port := 8080
	config := &InjectConf{ProxyPort: port}

	expectedEnvs := []corev1.EnvVar{
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
			Value: strconv.Itoa(port),
		},
		{
			Name:  ProxyEnvName,
			Value: "http://$(" + NodeNameEnvName + "):$(" + ProxyPortEnvName + ")",
		},
	}

	actualEnvs := envsFromConfig(config)

	assert.Equal(t, expectedEnvs, actualEnvs)
}
