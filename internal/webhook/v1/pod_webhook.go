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
// +kubebuilder:rbac:groups="",resources=namespaces;pods,verbs=get;list;watch
package v1

import (
	"context"
	"fmt"

	"d7y.io/dragonfly-p2p-webhook/internal/webhook/v1/injector"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// nolint:unused
// log is for logging in this package.
var podlog = logf.Log.WithName("pod-resource")

// SetupPodWebhookWithManager registers the webhook for Pod in the manager.
func SetupPodWebhookWithManager(mgr ctrl.Manager) error {
	configManager := injector.NewConfigManager(injector.InjectConfigMapPath)
	if err := mgr.Add(configManager); err != nil {
		return fmt.Errorf("failed to add config manager to manager: %w", err)
	}

	defaulter := NewPodCustomDefaulter(mgr.GetClient(), configManager)

	return ctrl.NewWebhookManagedBy(mgr).For(&corev1.Pod{}).
		WithDefaulter(defaulter).
		Complete()
}

type Injector interface {
	Inject(pod *corev1.Pod, config *injector.InjectConf)
}

// +kubebuilder:webhook:path=/mutate--v1-pod,mutating=true,failurePolicy=fail,sideEffects=None,groups="",resources=pods,verbs=create;update,versions=v1,name=mpod-v1.d7y.io,admissionReviewVersions=v1

// PodCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind Pod when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type PodCustomDefaulter struct {
	configManager *injector.ConfigManager
	kubeClient    client.Client
	injectors     []Injector
}

var _ webhook.CustomDefaulter = &PodCustomDefaulter{}

func NewPodCustomDefaulter(c client.Client, configManager *injector.ConfigManager) *PodCustomDefaulter {
	return &PodCustomDefaulter{
		kubeClient:    c,
		configManager: configManager,
		injectors: []Injector{
			injector.NewProxyEnvInjector(),
			injector.NewUnixSocketInjector(),
			injector.NewToolsInitcontainerInjector(),
		},
	}
}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind Pod.
func (d *PodCustomDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	pod, ok := obj.(*corev1.Pod)

	if !ok {
		return fmt.Errorf("expected an Pod object but got %T", obj)
	}
	podlog.Info("Defaulting for Pod", "name", pod.GetName())

	d.applyDefaults(ctx, pod)
	return nil
}

func (d *PodCustomDefaulter) applyDefaults(ctx context.Context, pod *corev1.Pod) {
	config := d.configManager.GetConfig()
	// check if need inject
	if !d.injectRequired(ctx, pod) {
		podlog.Info("Pod not inject", "name", pod.GetName())
		return
	}
	podlog.Info("Pod inject ")
	for _, ij := range d.injectors {
		ij.Inject(pod, config)
	}
}

func (d *PodCustomDefaulter) injectRequired(ctx context.Context, pod *corev1.Pod) bool {
	podlog.Info("func injectRequired start")
	return d.isNamespaceInjectionEnabled(ctx, pod) || d.isPodInjectionEnabled(ctx, pod)
}

func (d *PodCustomDefaulter) isNamespaceInjectionEnabled(ctx context.Context, pod *corev1.Pod) bool {
	podlog.Info("func injectNamespace get pod namespace", "pod", pod.Name)
	nsName := pod.GetNamespace()
	ns := &corev1.Namespace{}
	if err := d.kubeClient.Get(ctx, client.ObjectKey{Name: nsName}, ns); err != nil {
		podlog.Error(err, "failed to get namespace", "namespace", nsName)
		return false
	}

	labels := ns.GetLabels()
	podlog.Info(
		"func injectNamespace pod namespace lables",
		"pod", pod.Name,
		"labels", labels,
	)
	if len(labels) == 0 {
		podlog.Info(
			"namespace missing required injection label",
			"namespace", nsName,
			"requiredLabel", injector.NamespaceInjectLabelName,
			"pod", pod.Name,
		)
		return false
	}

	if v, ok := labels[injector.NamespaceInjectLabelName]; !ok ||
		v != injector.NamespaceInjectLabelValue {
		podlog.Info(
			"Namespace skipped injection: label not enabled",
			"namespace", nsName,
			"label", fmt.Sprintf("%s: %s", injector.NamespaceInjectLabelName, v),
			"pod", pod.Name,
		)
		return false
	}
	podlog.Info(
		"func injectNamespace check success",
		"namespace", nsName,
		"labels", labels,
		"pod", pod.Name,
	)
	return true
}

func (d *PodCustomDefaulter) isPodInjectionEnabled(_ context.Context, pod *corev1.Pod) bool {
	podlog.Info("func injectPod start", "pod", pod.Name)
	annotations := pod.GetAnnotations()
	if len(annotations) == 0 {
		podlog.Info(
			"pod missing required injection annotation, skip inject",
			"pod", pod.Name,
			"annotation", injector.PodInjectAnnotationName,
		)
		return false
	}

	podlog.Info(
		"func injectPod get annotations",
		"pod", pod.Name,
		"annotations", annotations,
	)
	if v, ok := annotations[injector.PodInjectAnnotationName]; !ok ||
		v != injector.PodInjectAnnotationValue {
		podlog.Info(
			"pod skipped injection: annotation not true, skip inject",
			"pod", pod.Name,
			"annotation", injector.PodInjectAnnotationName,
		)
		return false
	}
	podlog.Info("func injectPod success", "pod", pod.Name)
	return true
}
