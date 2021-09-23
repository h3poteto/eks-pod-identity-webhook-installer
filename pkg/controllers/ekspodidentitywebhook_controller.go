/*
Copyright 2021.

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
	"reflect"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	installerv1alpha1 "github.com/h3poteto/eks-pod-identity-webhook-installer/api/v1alpha1"
)

// EKSPodIdentityWebhookReconciler reconciles a EKSPodIdentityWebhook object
type EKSPodIdentityWebhookReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Logger   logr.Logger
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=installer.h3poteto.dev,resources=ekspodidentitywebhooks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=installer.h3poteto.dev,resources=ekspodidentitywebhooks/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=installer.h3poteto.dev,resources=ekspodidentitywebhooks/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the EKSPodIdentityWebhook object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *EKSPodIdentityWebhookReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = r.Logger.WithValues("EKSPodIdentityWebhook", req.NamespacedName)
	_ = log.FromContext(ctx)

	r.Logger.Info("Fetching EKSPodIdentityWebhook resources", "Namespace", req.Namespace, "Name", req.Name)
	resource := installerv1alpha1.EKSPodIdentityWebhook{}
	if err := r.Client.Get(ctx, req.NamespacedName, &resource); err != nil {
		r.Logger.Info("Failed to get EKSPodIdentityWebhook resources", "error", err)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if err := r.syncEKSPodIdentityWebhook(ctx, &resource); err != nil {
		r.Recorder.Eventf(&resource, corev1.EventTypeWarning, "Error", "Failed to sync: %v", err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EKSPodIdentityWebhookReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&installerv1alpha1.EKSPodIdentityWebhook{}).
		Complete(r)
}

func (r *EKSPodIdentityWebhookReconciler) syncEKSPodIdentityWebhook(ctx context.Context, resource *installerv1alpha1.EKSPodIdentityWebhook) error {
	r.Logger.Info("Syncing", "Namespace", resource.Namespace, "Name", resource.Name)

	newResource := resource.DeepCopy()
	if err := r.syncSecrets(ctx, newResource); err != nil {
		return err
	}

	if !reflect.DeepEqual(resource.Status, newResource.Status) {
		if err := r.Client.Update(ctx, newResource); err != nil {
			r.Logger.Error(err, "Failed to update EKSPodIdentityWebhook", "Namespace", newResource.Namespace, "Name", newResource.Name)
			return err
		}
		r.Logger.Info("Success to update with Secret", "Namespace", newResource.Namespace, "Name", newResource.Name)
	}
	return nil
}

func (r *EKSPodIdentityWebhookReconciler) secretShouldCreate(ctx context.Context, ref *installerv1alpha1.SecretRef) (bool, *corev1.Secret, error) {
	if ref == nil {
		return true, nil, nil
	}
	secret := corev1.Secret{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: ref.Namespace}, &secret)
	if kerrors.IsNotFound(err) {
		return true, nil, nil
	} else if err != nil {
		r.Logger.Error(err, "Failed to get secret", "Namespace", ref.Namespace, "Name", ref.Name)
		return false, nil, err
	}
	return false, &secret, nil
}

func (r *EKSPodIdentityWebhookReconciler) syncSecrets(ctx context.Context, resource *installerv1alpha1.EKSPodIdentityWebhook) error {
	create, secret, err := r.secretShouldCreate(ctx, resource.Status.PodIdentityWebhookSecret)
	if err != nil {
		return err
	}
	if create {
		key, cert, err := newCertificates(resource.Name, resource.Spec.Namespace)
		if err != nil {
			r.Logger.Error(err, "Failed to generate certificate")
			return err
		}
		secret = generateSecret(resource, resource.Spec.Namespace, resource.Name, key, cert)
		if err := r.Client.Create(ctx, secret); err != nil {
			r.Logger.Error(err, "Failed to create Secret", "Namespace", secret.Namespace, "Name", secret.Name)
			return err
		}
		r.Logger.Info("Success to create Secret", "Namespace", secret.Namespace, "Name", secret.Name)
	}
	resource.Status.PodIdentityWebhookSecret = &installerv1alpha1.SecretRef{
		Namespace: secret.Namespace,
		Name:      secret.Name,
	}
	return nil
}
