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
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"d7y.io/dragonfly-injector/internal/webhook/v1/injector"
	"gopkg.in/yaml.v3"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// admissionDecodeScheme is the minimal scheme needed to decode a Pod out of an
// AdmissionReview, mirroring the scheme used by the webhook manager.
var admissionDecodeScheme = func() *runtime.Scheme {
	s := runtime.NewScheme()
	if err := corev1.AddToScheme(s); err != nil {
		panic(fmt.Sprintf("failed to add corev1 to the test scheme: %v", err))
	}
	return s
}()

var admissionDecoder = admission.NewDecoder(admissionDecodeScheme)

// decodeAdmissionReview replays the decoding steps the controller-runtime
// webhook server performs before it calls PodCustomDefaulter.Default: the
// AdmissionReview body is decoded, a request must be present, and the inlined
// object must decode into a *corev1.Pod. Malformed input is reported as an
// error instead of being passed on to the defaulter.
func decodeAdmissionReview(raw []byte) (*admissionv1.AdmissionRequest, *corev1.Pod, error) {
	review := &admissionv1.AdmissionReview{}
	if err := json.Unmarshal(raw, review); err != nil {
		return nil, nil, fmt.Errorf("failed to decode admission review: %w", err)
	}

	if review.Request == nil {
		return nil, nil, errors.New("admission review does not contain a request")
	}

	pod := &corev1.Pod{}
	if err := admissionDecoder.Decode(admission.Request{AdmissionRequest: *review.Request}, pod); err != nil {
		return review.Request, nil, fmt.Errorf("failed to decode admission request object: %w", err)
	}

	return review.Request, pod, nil
}

// admissionReviewForPod builds a realistic AdmissionReview payload for a Pod.
// The annotation used to request injection is only set when withInjection is
// true so that both the mutated and the untouched paths are covered.
func admissionReviewForPod(withInjection bool) []byte {
	annotations := map[string]string{}
	if withInjection {
		annotations[injector.PodInjectAnnotationName] = injector.PodInjectAnnotationValue
	}

	pod := corev1.Pod{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Pod"},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-pod",
			Namespace:   "default",
			Annotations: annotations,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "app", Image: "busybox:1.36"}},
		},
	}

	object, err := json.Marshal(pod)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal the seed pod: %v", err))
	}

	review := admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{APIVersion: "admission.k8s.io/v1", Kind: "AdmissionReview"},
		Request: &admissionv1.AdmissionRequest{
			UID:       "seed-uid",
			Kind:      metav1.GroupVersionKind{Version: "v1", Kind: "Pod"},
			Resource:  metav1.GroupVersionResource{Version: "v1", Resource: "pods"},
			Namespace: "default",
			Name:      "test-pod",
			Operation: admissionv1.Create,
			Object:    runtime.RawExtension{Raw: object},
		},
	}

	raw, err := json.Marshal(review)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal the seed admission review: %v", err))
	}

	return raw
}

// FuzzAdmissionReviewDecode feeds arbitrary bytes through the AdmissionReview
// decode path. A malformed payload must be turned into an error and must never
// panic, and anything that decodes into a Pod must survive the defaulting
// logic of the webhook.
func FuzzAdmissionReviewDecode(f *testing.F) {
	seeds := [][]byte{
		admissionReviewForPod(true),
		admissionReviewForPod(false),
		[]byte(`{"apiVersion":"admission.k8s.io/v1","kind":"AdmissionReview","request":null}`),
		[]byte(`{"apiVersion":"admission.k8s.io/v1","kind":"AdmissionReview","request":{"uid":"empty-object","object":null}}`),
		[]byte(`{"apiVersion":"admission.k8s.io/v1","kind":"AdmissionReview","request":{"uid":"bad-raw","object":{"raw":"!!!"}}}`),
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	configDir := f.TempDir()
	config, err := yaml.Marshal(injector.DefaultConfig())
	if err != nil {
		f.Fatalf("failed to marshal the test config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "injector.yaml"), config, 0o644); err != nil {
		f.Fatalf("failed to write the test config: %v", err)
	}

	namespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "default"}}
	kubeClient := fake.NewClientBuilder().WithScheme(admissionDecodeScheme).WithObjects(namespace).Build()
	defaulter := NewPodCustomDefaulter(kubeClient, injector.NewConfigManager(configDir))

	f.Fuzz(func(t *testing.T, raw []byte) {
		// Reaching this line at all is the "does not panic" assertion: a panic
		// would abort the fuzz target with a crash report instead of an error.
		request, pod, err := decodeAdmissionReview(raw)
		if err != nil {
			if pod != nil {
				t.Fatalf("decode returned an error (%v) together with a pod", err)
			}
			return
		}

		if request == nil || pod == nil {
			t.Fatal("decode succeeded without returning both the request and the pod")
		}

		if err := defaulter.Default(context.Background(), pod); err != nil {
			t.Fatalf("defaulting a decoded Pod failed: %v", err)
		}
	})
}
