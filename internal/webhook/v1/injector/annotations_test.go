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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Pod injection annotations", func() {
	It("defaults opt-in annotations to false", func() {
		pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pod"}}
		Expect(injectInitContainersEnabled(pod)).To(BeFalse())
		Expect(binariesInitFirstEnabled(pod)).To(BeFalse())
	})

	It("enables features only when the annotation value is true", func() {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					InjectInitContainersAnnotationName: "false",
					BinariesInitFirstAnnotationName:    "yes",
				},
			},
		}
		Expect(injectInitContainersEnabled(pod)).To(BeFalse())
		Expect(binariesInitFirstEnabled(pod)).To(BeFalse())
	})

	It("enables features when the annotation value is true", func() {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					InjectInitContainersAnnotationName: InjectInitContainersAnnotationValue,
					BinariesInitFirstAnnotationName:    BinariesInitFirstAnnotationValue,
				},
			},
		}
		Expect(injectInitContainersEnabled(pod)).To(BeTrue())
		Expect(binariesInitFirstEnabled(pod)).To(BeTrue())
	})
})
