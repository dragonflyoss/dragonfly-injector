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

import corev1 "k8s.io/api/core/v1"

// podAnnotationEnabled checks if the pod annotation key is set to value.
func podAnnotationEnabled(annotations map[string]string, key, value string) bool {
	v, ok := annotations[key]
	return ok && v == value
}

// skipUnixSockInjectEnabled checks if the dfdaemon socket mount should be skipped.
func skipUnixSockInjectEnabled(pod *corev1.Pod) bool {
	return podAnnotationEnabled(pod.Annotations, SkipUnixSockInjectAnnotationName, SkipUnixSockInjectAnnotationValue)
}

// injectInitContainersEnabled checks if mounts should also be added to init containers.
func injectInitContainersEnabled(pod *corev1.Pod) bool {
	return podAnnotationEnabled(pod.Annotations, InjectInitContainersAnnotationName, InjectInitContainersAnnotationValue)
}

// binariesInitFirstEnabled checks if dragonfly-binaries should be inserted first.
func binariesInitFirstEnabled(pod *corev1.Pod) bool {
	return podAnnotationEnabled(pod.Annotations, BinariesInitFirstAnnotationName, BinariesInitFirstAnnotationValue)
}

// forEachNonInstallerInitContainer applies fn to each init container except the
// dragonfly-binaries installer, but only when inject-init-containers is enabled.
func forEachNonInstallerInitContainer(pod *corev1.Pod, fn func(*corev1.Container)) {
	if !injectInitContainersEnabled(pod) {
		return
	}

	for i := range pod.Spec.InitContainers {
		if pod.Spec.InitContainers[i].Name == InitContainerName {
			continue
		}
		fn(&pod.Spec.InitContainers[i])
	}
}
