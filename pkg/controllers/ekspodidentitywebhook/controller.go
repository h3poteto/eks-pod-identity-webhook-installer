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

package ekspodidentitywebhook

import (
	"context"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	installerv1alpha1 "github.com/h3poteto/eks-pod-identity-webhook-installer/api/v1alpha1"
	"github.com/h3poteto/eks-pod-identity-webhook-installer/pkg/generator"
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
//+kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services;serviceaccounts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles;rolebindings;clusterroles;clusterrolebindings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="admissionregistration.k8s.io",resources=mutatingwebhookconfigurations,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

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

	generator.Namespace = resource.Spec.Namespace

	newResource := resource.DeepCopy()

	create, serviceAccount, err := r.serviceAccountShouldCreate(ctx, newResource.Status.PodIdentityWebhookServiceAccount)
	if err != nil {
		return err
	}
	if create {
		serviceAccount, err = r.createServiceAccount(ctx, newResource)
		if err != nil {
			return err
		}
	}
	{
		newResource.Status.PodIdentityWebhookServiceAccount = &installerv1alpha1.ServiceAccountRef{
			Namespace: serviceAccount.Namespace,
			Name:      serviceAccount.Name,
		}
		if !reflect.DeepEqual(resource.Status, newResource.Status) {
			if err := r.Client.Status().Update(ctx, newResource); err != nil {
				r.Logger.Error(err, "Failed to update EKSPodIdentityWebhook")
				return err
			}
			r.Logger.Info("Success to update")
			return nil
		}
	}

	create, service, err := r.serviceShouldCreate(ctx, newResource.Status.PodIdentityWebhookService)
	if err != nil {
		return err
	}
	if create {
		service, err = r.createService(ctx, newResource)
		if err != nil {
			return err
		}
	}
	{
		newResource.Status.PodIdentityWebhookService = &installerv1alpha1.ServiceRef{
			Namespace: service.Namespace,
			Name:      service.Name,
		}
		if !reflect.DeepEqual(resource.Status, newResource.Status) {
			if err := r.Client.Status().Update(ctx, newResource); err != nil {
				r.Logger.Error(err, "Failed to update EKSPodIdentityWebhook")
				return err
			}
			r.Logger.Info("Success to update")
			return nil
		}
	}

	create, daemonset, err := r.daemonsetShouldCreate(ctx, newResource.Status.PodIdentityWebhookDaemonset)
	if err != nil {
		return err
	}
	if create {
		daemonset, err = r.createDaemonset(ctx, newResource)
		if err != nil {
			return err
		}
	}
	{
		newResource.Status.PodIdentityWebhookDaemonset = &installerv1alpha1.DaemonsetRef{
			Namespace: daemonset.Namespace,
			Name:      daemonset.Name,
		}
		if !reflect.DeepEqual(resource.Status, newResource.Status) {
			if err := r.Client.Status().Update(ctx, newResource); err != nil {
				r.Logger.Error(err, "Failed to update EKSPodIdentityWebhook")
				return err
			}
			r.Logger.Info("Success to update")
			return nil
		}
	}

	create, mutating, err := r.mutatingWebhookConfigurationShouldCreate(ctx, newResource.Status.PodIdentityWebhookConfiguration)
	if err != nil {
		return err
	}
	if create {
		mutating, err = r.createMutatingWebhookConfiguration(ctx, newResource, service)
		if err != nil {
			return err
		}
	}
	{
		newResource.Status.PodIdentityWebhookConfiguration = &installerv1alpha1.MutatingWebhookConfigurationRef{
			Name: mutating.Name,
		}
		if !reflect.DeepEqual(resource.Status, newResource.Status) {
			if err := r.Client.Status().Update(ctx, newResource); err != nil {
				r.Logger.Error(err, "Failed to update EKSPodIdentityWebhook")
				return err
			}
			r.Logger.Info("Success to update")
			return nil
		}
	}
	r.Logger.Info("Already synced")
	return nil
}

func (r *EKSPodIdentityWebhookReconciler) createServiceAccount(ctx context.Context, resource *installerv1alpha1.EKSPodIdentityWebhook) (*corev1.ServiceAccount, error) {
	serviceAccount := generator.GenerateServiceAccount(resource)

	{
		exists := corev1.ServiceAccount{}
		err := r.Client.Get(ctx, types.NamespacedName{Namespace: serviceAccount.Namespace, Name: serviceAccount.Name}, &exists)
		if kerrors.IsNotFound(err) {
			if err := r.Client.Create(ctx, serviceAccount); err != nil {
				r.Logger.Error(err, "Failed to create ServiceAccount")
				return nil, err
			}
			r.Logger.Info("Success to create ServiceAccount")
		} else {
			exists.SetOwnerReferences(serviceAccount.OwnerReferences)
			if err := r.Client.Update(ctx, &exists); err != nil {
				r.Logger.Error(err, "Failed to update ServiceAccount")
				return nil, err
			}
			r.Logger.Info("Success to update ServiceAccount")
		}
	}

	role := generator.GenerateRole(resource)
	{
		exists := rbacv1.Role{}
		err := r.Client.Get(ctx, types.NamespacedName{Namespace: role.Namespace, Name: role.Name}, &exists)
		if kerrors.IsNotFound(err) {
			if err := r.Client.Create(ctx, role); err != nil {
				r.Logger.Error(err, "Failed to create Role")
				return nil, err
			}
			r.Logger.Info("Success to create Role")
		} else {
			exists.SetOwnerReferences(role.OwnerReferences)
			exists.Rules = role.Rules
			if err := r.Client.Update(ctx, &exists); err != nil {
				r.Logger.Error(err, "Failed to update Role")
				return nil, err
			}
			r.Logger.Info("Success to update Role")
		}
	}

	roleBinding := generator.GenerateRoleBinding(resource, role, serviceAccount)
	{
		exists := rbacv1.RoleBinding{}
		err := r.Client.Get(ctx, types.NamespacedName{Namespace: roleBinding.Namespace, Name: roleBinding.Name}, &exists)
		if kerrors.IsNotFound(err) {
			if err := r.Client.Create(ctx, roleBinding); err != nil {
				r.Logger.Error(err, "Failed to create RoleBinding")
				return nil, err
			}
			r.Logger.Info("Success to create RoleBinding")
		} else {
			exists.SetOwnerReferences(roleBinding.OwnerReferences)
			exists.Subjects = roleBinding.Subjects
			exists.RoleRef = roleBinding.RoleRef
			if err := r.Client.Update(ctx, &exists); err != nil {
				r.Logger.Error(err, "Failed to update RoleBinding")
				return nil, err
			}
			r.Logger.Info("Success to update RoleBinding")
		}

	}

	clusterRole := generator.GenerateClusterRole(resource)
	{
		exists := rbacv1.ClusterRole{}
		err := r.Client.Get(ctx, types.NamespacedName{Name: clusterRole.Name}, &exists)
		if kerrors.IsNotFound(err) {
			if err := r.Client.Create(ctx, clusterRole); err != nil {
				r.Logger.Error(err, "Failed to create ClusterRole")
				return nil, err
			}
			r.Logger.Info("Success to create ClusterRole")
		} else {
			exists.SetOwnerReferences(clusterRole.OwnerReferences)
			exists.Rules = clusterRole.Rules
			if err := r.Client.Update(ctx, &exists); err != nil {
				r.Logger.Error(err, "Failed to update ClusterRole")
				return nil, err
			}
			r.Logger.Info("Success to update ClusterRole")
		}

	}

	clusterRoleBinding := generator.GenerateClusterRoleBinding(resource, clusterRole, serviceAccount)
	{
		exists := rbacv1.ClusterRoleBinding{}
		err := r.Client.Get(ctx, types.NamespacedName{Name: clusterRoleBinding.Name}, &exists)
		if kerrors.IsNotFound(err) {
			if err := r.Client.Create(ctx, clusterRoleBinding); err != nil {
				r.Logger.Error(err, "Failed to create ClusterRoleBinding")
				return nil, err
			}
			r.Logger.Info("Success to create ClusterRoleBinding")
		} else {
			exists.SetOwnerReferences(clusterRoleBinding.OwnerReferences)
			exists.Subjects = clusterRoleBinding.Subjects
			exists.RoleRef = clusterRoleBinding.RoleRef
			if err := r.Client.Update(ctx, &exists); err != nil {
				r.Logger.Error(err, "failed to update ClusterRoleBinding")
				return nil, err
			}
			r.Logger.Info("Success to update ClusterRoleBinding")
		}

	}
	return serviceAccount, nil
}

func (r *EKSPodIdentityWebhookReconciler) createDaemonset(ctx context.Context, resource *installerv1alpha1.EKSPodIdentityWebhook) (*appsv1.DaemonSet, error) {
	daemonset := generator.GenerateDaemonset(resource)
	if err := r.Client.Create(ctx, daemonset); err != nil {
		r.Logger.Error(err, "failed to create DaemonSet")
		return nil, err
	}
	r.Logger.Info("Success to create DaemonSet")
	return daemonset, nil
}

func (r *EKSPodIdentityWebhookReconciler) createService(ctx context.Context, resource *installerv1alpha1.EKSPodIdentityWebhook) (*corev1.Service, error) {
	service := generator.GenerateService(resource)
	if err := r.Client.Create(ctx, service); err != nil {
		r.Logger.Error(err, "Failed to create Service")
		return nil, err
	}
	r.Logger.Info("Success to create Service")
	return service, nil
}

func (r *EKSPodIdentityWebhookReconciler) createMutatingWebhookConfiguration(ctx context.Context, resource *installerv1alpha1.EKSPodIdentityWebhook, service *corev1.Service) (*admissionregistrationv1.MutatingWebhookConfiguration, error) {
	// Get default service account token, and use the CA.
	// https://github.com/aws/amazon-eks-pod-identity-webhook/blob/35a57cc479ae760760bfa9b5a628a488a46adad2/hack/webhook-patch-ca-bundle.sh#L10-L19
	defaultSA := corev1.ServiceAccount{}
	if err := r.Client.Get(ctx, types.NamespacedName{Name: "default", Namespace: resource.Spec.Namespace}, &defaultSA); err != nil {
		r.Logger.Error(err, "Failed to get default service account")
		return nil, err
	}
	if len(defaultSA.Secrets) != 1 {
		return nil, fmt.Errorf("%s/%s has invalid secrets", defaultSA.Namespace, defaultSA.Name)
	}
	defaultSAToken := corev1.Secret{}
	if err := r.Client.Get(ctx, types.NamespacedName{Name: defaultSA.Secrets[0].Name, Namespace: resource.Spec.Namespace}, &defaultSAToken); err != nil {
		r.Logger.Error(err, "Failed to get default service account token")
		return nil, err
	}
	CA := defaultSAToken.Data["ca.crt"]

	mutating := generator.GenerateMutatingWebhookConfiguration(resource, service, CA)
	if err := r.Client.Create(ctx, mutating); err != nil {
		r.Logger.Error(err, "Failed to create MutatingWebhookConfiguration")
		return nil, err
	}
	r.Logger.Info("Success to create MutatingWebhookConfiguration")
	return mutating, nil
}

func (r *EKSPodIdentityWebhookReconciler) serviceAccountShouldCreate(ctx context.Context, ref *installerv1alpha1.ServiceAccountRef) (bool, *corev1.ServiceAccount, error) {
	if ref == nil {
		return true, nil, nil
	}
	serviceAccount := corev1.ServiceAccount{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: ref.Namespace}, &serviceAccount)
	if kerrors.IsNotFound(err) {
		return true, nil, nil
	} else if err != nil {
		r.Logger.Error(err, "Failed to get serviceaccount")
		return false, nil, err
	}
	return false, &serviceAccount, nil
}

func (r *EKSPodIdentityWebhookReconciler) daemonsetShouldCreate(ctx context.Context, ref *installerv1alpha1.DaemonsetRef) (bool, *appsv1.DaemonSet, error) {
	if ref == nil {
		return true, nil, nil
	}
	daemonset := appsv1.DaemonSet{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: ref.Namespace}, &daemonset)
	if kerrors.IsNotFound(err) {
		return true, nil, nil
	} else if err != nil {
		r.Logger.Error(err, "Failed to get daemonset", "Namespace", ref.Namespace, "Name", ref.Name)
		return false, nil, err
	}
	return false, &daemonset, nil
}

func (r *EKSPodIdentityWebhookReconciler) serviceShouldCreate(ctx context.Context, ref *installerv1alpha1.ServiceRef) (bool, *corev1.Service, error) {
	if ref == nil {
		return true, nil, nil
	}
	service := corev1.Service{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: ref.Namespace}, &service)
	if kerrors.IsNotFound(err) {
		return true, nil, nil
	} else if err != nil {
		r.Logger.Error(err, "Failed to get service")
		return false, nil, err
	}
	return false, &service, nil
}

func (r *EKSPodIdentityWebhookReconciler) mutatingWebhookConfigurationShouldCreate(
	ctx context.Context,
	ref *installerv1alpha1.MutatingWebhookConfigurationRef,
) (bool, *admissionregistrationv1.MutatingWebhookConfiguration, error) {
	if ref == nil {
		return true, nil, nil
	}
	mutating := admissionregistrationv1.MutatingWebhookConfiguration{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: ref.Name}, &mutating)
	if kerrors.IsNotFound(err) {
		return true, nil, nil
	} else if err != nil {
		r.Logger.Error(err, "Failed to get mutating")
		return false, nil, err
	}
	return false, &mutating, nil
}
