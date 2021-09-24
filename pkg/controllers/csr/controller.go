package csr

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/h3poteto/eks-pod-identity-webhook-installer/pkg/generator"
	certificatesv1 "k8s.io/api/certificates/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type CSRReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Logger   logr.Logger
	Recorder record.EventRecorder
}

// TODO: rbac

func (r *CSRReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.Logger.Info("Fetching CertificateSigningRequest resources", "Namespace", req.Namespace, "Name", req.Name)
	resource := certificatesv1.CertificateSigningRequest{}
	if err := r.Client.Get(ctx, req.NamespacedName, &resource); err != nil {
		r.Logger.Info("Failed to get CertificateSigningRequest resources", "error", err)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if err := r.approveCSR(ctx, &resource); err != nil {
		r.Recorder.Eventf(&resource, corev1.EventTypeWarning, "Error", "Failed to sync: %v", err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *CSRReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&certificatesv1.CertificateSigningRequest{}).
		// Watches(&source.Kind{Type: &certificatesv1.CertificateSigningRequest{}}, &handler.EnqueueRequestForObject{}).
		Complete(r)
}

func (r *CSRReconciler) approveCSR(ctx context.Context, resource *certificatesv1.CertificateSigningRequest) error {
	if generator.Namespace == "" {
		r.Logger.Info("Namespace is not set, so waiting for namespace")
		return nil
	}
	if resource.Spec.Username != "system:serviceaccount:"+generator.Namespace+":"+generator.ServiceAccountName {
		r.Logger.Info("CSR is not owned", resource.Namespace, resource.Name)
		return nil
	}

	for _, condition := range resource.Status.Conditions {
		if condition.Type == certificatesv1.CertificateApproved {
			r.Logger.Info("CSR is already approved", "Name", resource.Name)
			return nil
		}
		if condition.Type == certificatesv1.CertificateDenied {
			r.Logger.Info("CSR is already denied", "Name", resource.Name)
			return nil
		}
		if condition.Type == certificatesv1.CertificateFailed {
			r.Logger.Info("CSR is already failed", "Name", resource.Name)
			return nil
		}
	}

	rest := ctrl.GetConfigOrDie()
	client, err := clientset.NewForConfig(rest)
	if err != nil {
		return err
	}

	resource.Status.Conditions = append(resource.Status.Conditions, certificatesv1.CertificateSigningRequestCondition{
		Type:           certificatesv1.CertificateApproved,
		Status:         corev1.ConditionTrue,
		Reason:         "AutoApproved",
		Message:        "This CSR was approved by eks-pod-identity-webhook-installer",
		LastUpdateTime: metav1.Now(),
	})
	_, err = client.CertificatesV1().CertificateSigningRequests().UpdateApproval(ctx, resource.Name, resource, metav1.UpdateOptions{})
	if err != nil {
		r.Logger.Error(err, "Failed to update", "CSR", resource.Name)
		return err
	}
	r.Logger.Info("CertificateSigningRequest is approve", "Name", resource.Name)

	return nil
}
