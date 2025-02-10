/*
Copyright 2020 Humio https://humio.com

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

package controllers

import (
	"context"

	"k8s.io/apimachinery/pkg/api/meta"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	corev1alpha1 "github.com/humio/humio-operator/api/v1alpha1"
)

// HumioPdfRenderServiceReconciler reconciles a HumioPdfRenderService object
type HumioPdfRenderServiceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=core.humio.com,resources=humiopdfrenderservices,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core.humio.com,resources=humiopdfrenderservices/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core.humio.com,resources=humiopdfrenderservices/finalizers,verbs=update

// Reconcile implements the reconciliation logic for HumioPdfRenderService.
func (r *HumioPdfRenderServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the HumioPdfRenderService instance
	var humioPdfRenderService corev1alpha1.HumioPdfRenderService
	if err := r.Get(ctx, req.NamespacedName, &humioPdfRenderService); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.Info("Reconciling HumioPdfRenderService", "Name", humioPdfRenderService.Name, "Namespace", humioPdfRenderService.Namespace)

	// Update status before returning
	defer func() {
		if err := r.Status().Update(ctx, &humioPdfRenderService); err != nil {
			logger.Error(err, "Failed to update status")
		}
	}()

	// Handle deletion and finalizers if needed.
	if !humioPdfRenderService.ObjectMeta.DeletionTimestamp.IsZero() {
		// TODO: Add finalizer logic if required.
		logger.Info("HumioPdfRenderService is being deleted", "Name", humioPdfRenderService.Name)
		// Update available condition
		meta.SetStatusCondition(&humioPdfRenderService.Status.Conditions, metav1.Condition{
			Type:    "Available",
			Status:  metav1.ConditionTrue,
			Reason:  "ComponentsReady",
			Message: "All components deployed successfully",
		})

		meta.SetStatusCondition(&humioPdfRenderService.Status.Conditions, metav1.Condition{
			Type:    "Progressing",
			Status:  metav1.ConditionFalse,
			Reason:  "ReconciliationComplete",
			Message: "All resources reconciled successfully",
		})

		return ctrl.Result{}, nil
	}

	// Reconcile Deployment using controllerutil.CreateOrUpdate
	if err := r.reconcileDeployment(ctx, &humioPdfRenderService); err != nil {
		logger.Error(err, "Failed to reconcile Deployment")
		return ctrl.Result{}, err
	}

	// Reconcile Service using controllerutil.CreateOrUpdate
	if err := r.reconcileService(ctx, &humioPdfRenderService); err != nil {
		logger.Error(err, "Failed to reconcile Service")
		return ctrl.Result{}, err
	}

	// TODO: Reconcile Ingress and update CR status as needed.

	return ctrl.Result{}, nil
}

// constructDeployment builds the Deployment object for HumioPdfRenderService.
func (r *HumioPdfRenderServiceReconciler) constructDeployment(humioPdfRenderService *corev1alpha1.HumioPdfRenderService) *appsv1.Deployment {
	labels := map[string]string{
		"app":                           "humio-pdf-render-service", // TODO: Use a constant
		"humio-pdf-render-service-name": humioPdfRenderService.Name,
	}

	// Create a Deployment object with minimal objectMeta; spec will be set in CreateOrUpdate
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      humioPdfRenderService.Name + "-pdf-render-service", // Consider using a helper to generate the name.
			Namespace: humioPdfRenderService.Namespace,
		},
	}
	// Pre-set a default spec so that CreateOrUpdate always has a spec to work with.
	deployment.Spec = appsv1.DeploymentSpec{
		Replicas: &humioPdfRenderService.Spec.Replicas,
		Selector: &metav1.LabelSelector{
			MatchLabels: labels,
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels:      labels,
				Annotations: humioPdfRenderService.Spec.Annotations,
			},
			Spec: corev1.PodSpec{
				ServiceAccountName: humioPdfRenderService.Spec.ServiceAccountName,
				Affinity:           humioPdfRenderService.Spec.Affinity,
				Containers: []corev1.Container{{
					Name:            "humio-pdf-render-service", // TODO: Use a constant
					Image:           humioPdfRenderService.Spec.Image,
					ImagePullPolicy: corev1.PullIfNotPresent, // Make configurable if necessary.
					Resources:       humioPdfRenderService.Spec.Resources,
					Ports: []corev1.ContainerPort{{
						ContainerPort: humioPdfRenderService.Spec.Port,
						Name:          "http", // TODO: Use a constant
					}},
					Env:            humioPdfRenderService.Spec.Env,
					LivenessProbe:  humioPdfRenderService.Spec.LivenessProbe,
					ReadinessProbe: humioPdfRenderService.Spec.ReadinessProbe,
				}},
			},
		},
	}
	return deployment
}

// reconcileDeployment uses CreateOrUpdate to reconcile the Deployment.
func (r *HumioPdfRenderServiceReconciler) reconcileDeployment(ctx context.Context, humioPdfRenderService *corev1alpha1.HumioPdfRenderService) error {
	deployment := r.constructDeployment(humioPdfRenderService)
	// Set owner reference for garbage collection.
	if err := controllerutil.SetControllerReference(humioPdfRenderService, deployment, r.Scheme); err != nil {
		return err
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		labels := map[string]string{
			"app":                           "humio-pdf-render-service",
			"humio-pdf-render-service-name": humioPdfRenderService.Name,
		}
		deployment.Labels = labels
		deployment.Spec.Replicas = &humioPdfRenderService.Spec.Replicas
		deployment.Spec.Selector = &metav1.LabelSelector{
			MatchLabels: labels,
		}
		deployment.Spec.Template.ObjectMeta.Labels = labels
		deployment.Spec.Template.ObjectMeta.Annotations = humioPdfRenderService.Spec.Annotations
		deployment.Spec.Template.Spec.ServiceAccountName = humioPdfRenderService.Spec.ServiceAccountName
		deployment.Spec.Template.Spec.Affinity = humioPdfRenderService.Spec.Affinity
		deployment.Spec.Template.Spec.Containers = []corev1.Container{{
			Name:            "humio-pdf-render-service",
			Image:           humioPdfRenderService.Spec.Image,
			ImagePullPolicy: corev1.PullIfNotPresent,
			Resources:       humioPdfRenderService.Spec.Resources,
			Ports: []corev1.ContainerPort{{
				ContainerPort: humioPdfRenderService.Spec.Port,
				Name:          "http",
			}},
			Env:            humioPdfRenderService.Spec.Env,
			LivenessProbe:  humioPdfRenderService.Spec.LivenessProbe,
			ReadinessProbe: humioPdfRenderService.Spec.ReadinessProbe,
		}}
		return nil
	})
	return err
}

// constructService builds the Service object for HumioPdfRenderService.
func (r *HumioPdfRenderServiceReconciler) constructService(humioPdfRenderService *corev1alpha1.HumioPdfRenderService) *corev1.Service {
	labels := map[string]string{
		"app":                           "humio-pdf-render-service", // TODO: Use a constant
		"humio-pdf-render-service-name": humioPdfRenderService.Name,
	}
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      humioPdfRenderService.Name + "-pdf-render-service", // Consider a helper for the name.
			Namespace: humioPdfRenderService.Namespace,
		},
	}
	service.Spec = corev1.ServiceSpec{
		Selector: labels,
		Type:     humioPdfRenderService.Spec.ServiceType,
		Ports: []corev1.ServicePort{{
			Port:       humioPdfRenderService.Spec.Port,
			TargetPort: intstr.FromInt(int(humioPdfRenderService.Spec.Port)),
			Name:       "http", // TODO: Use a constant
		}},
	}
	return service
}

// reconcileService uses CreateOrUpdate to reconcile the Service.
func (r *HumioPdfRenderServiceReconciler) reconcileService(ctx context.Context, humioPdfRenderService *corev1alpha1.HumioPdfRenderService) error {
	service := r.constructService(humioPdfRenderService)
	if err := controllerutil.SetControllerReference(humioPdfRenderService, service, r.Scheme); err != nil {
		return err
	}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, service, func() error {
		labels := map[string]string{
			"app":                           "humio-pdf-render-service",
			"humio-pdf-render-service-name": humioPdfRenderService.Name,
		}
		service.Labels = labels
		service.Spec.Selector = labels
		service.Spec.Type = humioPdfRenderService.Spec.ServiceType
		service.Spec.Ports = []corev1.ServicePort{{
			Port:       humioPdfRenderService.Spec.Port,
			TargetPort: intstr.FromInt(int(humioPdfRenderService.Spec.Port)),
			Name:       "http",
		}}
		return nil
	})
	return err
}

// SetupWithManager sets up the controller with the Manager.
func (r *HumioPdfRenderServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1alpha1.HumioPdfRenderService{}).
		Complete(r)
}
