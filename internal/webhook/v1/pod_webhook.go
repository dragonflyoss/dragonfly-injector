/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// nolint:unused
// log is for logging in this package.
var podlog = logf.Log.WithName("pod-resource")

// SetupPodWebhookWithManager registers the webhook for Pod in the manager.
func SetupPodWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&corev1.Pod{}).
		WithDefaulter(NewPodCustomDefaulter()).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate--v1-pod,mutating=true,failurePolicy=fail,sideEffects=None,groups="",resources=pods,verbs=create;update,versions=v1,name=mpod-v1.d7y.io,admissionReviewVersions=v1

// PodCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind Pod when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type PodCustomDefaulter struct {
	inject_anotation string
	env_name         string
}

func NewPodCustomDefaulter() *PodCustomDefaulter {
	return &PodCustomDefaulter{
		inject_anotation: "dragonfly.io/inject",
		env_name:         "TEST",
	}
}

var _ webhook.CustomDefaulter = &PodCustomDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind Pod.
func (d *PodCustomDefaulter) Default(_ context.Context, obj runtime.Object) error {
	pod, ok := obj.(*corev1.Pod)

	if !ok {
		return fmt.Errorf("expected an Pod object but got %T", obj)
	}
	podlog.Info("Defaulting for Pod", "name", pod.GetName())

	d.applyDefaults(pod)
	return nil
}

func (d *PodCustomDefaulter) applyDefaults(pod *corev1.Pod) {
	// check if have annotations
	annotations := pod.GetAnnotations()

	if annotations == nil || annotations[d.inject_anotation] != "true" {
		podlog.Info(
			"Annotation  d.inject_anotation not found",
			"d.inject_annotation",
			d.inject_anotation,
			"annotations",
			annotations,
		)
		return
	}
	podlog.Info("Start applyDefaults")
	d.applyEnv(pod)
	d.applyUnixSocket(pod)
	d.applyInitContainer(pod)
}
func (d *PodCustomDefaulter) applyEnv(pod *corev1.Pod) {
	podlog.Info("applyEnv")
	// 希望设置一个Test环境变量
	containers := pod.Spec.Containers
	podlog.Info("Containers", "containers", containers)
	for i := range containers {
		c := &containers[i]
		d.applyContainerEnv(c)
	}
}

func (d *PodCustomDefaulter) applyUnixSocket(pod *corev1.Pod) {

}

func (d *PodCustomDefaulter) applyInitContainer(pod *corev1.Pod) {

}

func (d *PodCustomDefaulter) applyContainerEnv(c *corev1.Container) {
	podlog.Info("Container applyContainerEnv", "name", c.Name)
	for _, e := range c.Env {
		if e.Name == d.env_name {
			podlog.Info("Container has env", "name", c.Name)
			return
		}
	}
	podlog.Info("Container has env", "env", c.Env)
	c.Env = append(c.Env, corev1.EnvVar{
		Name:  d.env_name,
		Value: d.env_name,
	})
}
