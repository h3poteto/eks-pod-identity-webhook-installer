package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EKSPodIdentityWebhookSpec defines the desired state of EKSPodIdentityWebhook
type EKSPodIdentityWebhookSpec struct {

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type:=string
	TokenAudience string `json:"tokenAudience"`
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type:=string
	// +kubebuilder:default=default
	Namespace string `json:"namespace"`
}

// EKSPodIdentityWebhookStatus defines the observed state of EKSPodIdentityWebhook
type EKSPodIdentityWebhookStatus struct {
	// +nullable
	PodIdentityWebhookSecret *SecretRef `json:"podIdentityWebhookSecret,omitempty"`
	// +nullable
	PodIdentityWebhookService *ServiceRef `json:"podIdentityWebhookService,omitempty"`
	// +nullable
	PodIdentityWebhookDaemonset *DaemonsetRef `json:"podIdentityWebhookDaemonset,omitempty"`
	// +nullable
	PodIdentityWebhookConfiguration *MutatingWebhookConfigurationRef `json:"podIdentityWebhookConfiguration,omitempty"`
	// +nullable
	PodIdentityWebhookServiceAccount *ServiceAccountRef `json:"podIdentityWebhookServiceAccount,omitempty"`
	// +kubebuilder:default=init
	Phase string `json:"phase"`
}

type SecretRef Ref
type ServiceRef Ref
type DaemonsetRef Ref
type ServiceAccountRef Ref
type MutatingWebhookConfigurationRef struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=string
	Name string `json:"name"`
}

type Ref struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=string
	Namespace string `json:"namespace"`
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=string
	Name string `json:"name"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:scope=Cluster
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`

// EKSPodIdentityWebhook is the Schema for the ekspodidentitywebhooks API
type EKSPodIdentityWebhook struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EKSPodIdentityWebhookSpec   `json:"spec,omitempty"`
	Status EKSPodIdentityWebhookStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// EKSPodIdentityWebhookList contains a list of EKSPodIdentityWebhook
type EKSPodIdentityWebhookList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EKSPodIdentityWebhook `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EKSPodIdentityWebhook{}, &EKSPodIdentityWebhookList{})
}
