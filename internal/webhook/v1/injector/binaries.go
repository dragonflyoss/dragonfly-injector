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

// InjectedBinaries is the set of dragonfly client binaries installed by
// the init container and mounted into the application containers.
var InjectedBinaries = []string{
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

	initContainerCmd := append([]string{
		"install",
		"-D",
	},
		b.sourcePaths()...,
	)
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
		b.injectContainerVolumeMounts(&pod.Spec.Containers[i])
	}
}

// sourcePaths returns the in-image source paths of the injected binaries.
func (b *Binaries) sourcePaths() []string {
	paths := make([]string, 0, len(InjectedBinaries))
	for _, name := range InjectedBinaries {
		paths = append(paths, filepath.Join(BinaryDirPath, name))
	}
	return paths
}

// injectContainerVolumeMounts ensures each binary has its volume mount in
// the container. Mounts are checked per binary, so a container that only
// received a subset of the mounts (e.g. from a partial or older injection)
// is brought back to the full set without duplicating the existing ones.
func (b *Binaries) injectContainerVolumeMounts(container *corev1.Container) {
	if !hasVolumeMountNamed(container, BinaryVolumeName, BinaryVolumeMountPath) {
		container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
			Name:      BinaryVolumeName,
			MountPath: BinaryVolumeMountPath,
		})
	}

	for _, name := range InjectedBinaries {
		mountPath := filepath.Join(BinaryMountDirPath, name)
		if !hasVolumeMountNamed(container, BinaryVolumeName, mountPath) {
			container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
				Name:      BinaryVolumeName,
				MountPath: mountPath,
				SubPath:   name,
				ReadOnly:  true,
			})
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
