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

package injector

import (
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
)

var binaryNames = []string{
	DfgetBinaryName,
	DfcacheBinaryName,
	DfstoreBinaryName,
	DfctlBinaryName,
	DfdaemonBinaryName,
}

type Binaries struct{}

func NewBinaries() *Binaries {
	return &Binaries{}
}

func (b *Binaries) Inject(pod *corev1.Pod, config *Config) {
	logger.Info("Binaries inject", "pod namespace", pod.GetNamespace(), "pod name", pod.GetName())

	initContainerCmd := []string{"install", "-D"}
	for _, bin := range binaryNames {
		initContainerCmd = append(initContainerCmd, filepath.Join(BinaryDirPath, bin))
	}
	initContainerCmd = append(initContainerCmd, "-t", BinaryVolumeMountPath+"/")

	// Override initContainerImage with the value from annotations if it exists.
	initContainerImage := config.GetInitContainerImageReference()
	annotations := pod.Annotations
	if annotations != nil {
		if image, ok := annotations[InitContainerImageAnnotation]; ok {
			initContainerImage = image
		}
	}

	// Mutate the pod spec to add the init container.
	if !b.hasInitContainer(pod) {
		toolContainer := &corev1.Container{
			Name:            InitContainerName,
			Image:           initContainerImage,
			ImagePullPolicy: config.InitContainerImage.PullPolicy,
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      BinaryVolumeName,
					MountPath: BinaryVolumeMountPath,
				},
			},
			Command: initContainerCmd,
		}
		pod.Spec.InitContainers = append(pod.Spec.InitContainers, *toolContainer)
		pod.Spec.ImagePullSecrets = append(pod.Spec.ImagePullSecrets, config.InitContainerImage.PullSecrets...)
	}

	if !b.hasVolume(pod) {
		toolsVolume := &corev1.Volume{
			Name: BinaryVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		}
		pod.Spec.Volumes = append(pod.Spec.Volumes, *toolsVolume)
	}

	for i := range pod.Spec.Containers {
		if !b.hasVolumeMount(&pod.Spec.Containers[i]) {
			pod.Spec.Containers[i].VolumeMounts = append(pod.Spec.Containers[i].VolumeMounts, corev1.VolumeMount{
				Name:      BinaryVolumeName,
				MountPath: BinaryVolumeMountPath,
			})

			for _, bin := range binaryNames {
				pod.Spec.Containers[i].VolumeMounts = append(pod.Spec.Containers[i].VolumeMounts, corev1.VolumeMount{
					Name:      BinaryVolumeName,
					MountPath: filepath.Join(BinaryMountDirPath, bin),
					SubPath:   bin,
					ReadOnly:  true,
				})
			}
		}
	}
}

// hasInitContainer checks if the pod has the init container.
func (b *Binaries) hasInitContainer(pod *corev1.Pod) bool {
	for _, c := range pod.Spec.InitContainers {
		if c.Name == InitContainerName {
			return true
		}
	}

	return false
}

// hasVolume checks if the pod has the volume.
func (b *Binaries) hasVolume(pod *corev1.Pod) bool {
	for _, v := range pod.Spec.Volumes {
		if v.Name == BinaryVolumeName {
			return true
		}
	}

	return false
}

// hasVolumeMount checks if the container has the volume mount.
func (b *Binaries) hasVolumeMount(c *corev1.Container) bool {
	for _, vm := range c.VolumeMounts {
		if vm.Name == BinaryVolumeName {
			return true
		}
	}

	return false
}
