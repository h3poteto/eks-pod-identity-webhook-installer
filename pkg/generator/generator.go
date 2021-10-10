// Refs: https://github.com/aws/amazon-eks-pod-identity-webhook/tree/master/deploy

package generator

import (
	installerv1alpha1 "github.com/h3poteto/eks-pod-identity-webhook-installer/api/v1alpha1"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	utilpointer "k8s.io/utils/pointer"
)

const (
	baseName                         = "pod-identity-webhook"
	ServiceAccountName               = baseName
	ServiceName                      = baseName
	SecretName                       = baseName
	DaemonsetName                    = baseName
	MutatingWebhookconfigurationName = baseName

	WebhookServerLabelKey      = "ekspodidentitywebhooks.installer.h3poteto.dev"
	WebhookServerLabelValuePod = "pod"
)

var Namespace string = ""

func GenerateMutatingWebhookConfiguration(resource *installerv1alpha1.EKSPodIdentityWebhook, service *corev1.Service, serverCertificate []byte) *admissionregistrationv1.MutatingWebhookConfiguration {
	ignore := admissionregistrationv1.Ignore
	allscopes := admissionregistrationv1.AllScopes
	equivalent := admissionregistrationv1.Equivalent
	sideeffect := admissionregistrationv1.SideEffectClassNone
	return &admissionregistrationv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: MutatingWebhookconfigurationName,
			Labels: map[string]string{
				WebhookServerLabelKey: "webhook-configuration",
				"kind":                "mutator",
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(resource, schema.GroupVersionKind{
					Group:   installerv1alpha1.GroupVersion.Group,
					Version: installerv1alpha1.GroupVersion.Version,
					Kind:    "EKSPodIdentityWebhook",
				}),
			},
		},
		Webhooks: []admissionregistrationv1.MutatingWebhook{
			{
				Name: service.Name + "." + service.Namespace + ".svc",
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					Service: &admissionregistrationv1.ServiceReference{
						Namespace: service.Namespace,
						Name:      service.Name,
						Path:      utilpointer.StringPtr("/mutate"),
					},
					CABundle: serverCertificate,
				},
				Rules: []admissionregistrationv1.RuleWithOperations{
					{
						Operations: []admissionregistrationv1.OperationType{
							"CREATE",
						},
						Rule: admissionregistrationv1.Rule{
							APIGroups:   []string{""},
							APIVersions: []string{"v1"},
							Resources:   []string{"pods"},
							Scope:       &allscopes,
						},
					},
				},
				FailurePolicy:           &ignore,
				MatchPolicy:             &equivalent,
				SideEffects:             &sideeffect,
				TimeoutSeconds:          utilpointer.Int32Ptr(30),
				AdmissionReviewVersions: []string{"v1beta1"},
			},
		},
	}
}

func GenerateService(resource *installerv1alpha1.EKSPodIdentityWebhook) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ServiceName,
			Namespace: Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(resource, schema.GroupVersionKind{
					Group:   installerv1alpha1.GroupVersion.Group,
					Version: installerv1alpha1.GroupVersion.Version,
					Kind:    "EKSPodIdentityWebhook",
				}),
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Protocol: corev1.ProtocolTCP,
					Port:     443,
					TargetPort: intstr.IntOrString{
						IntVal: 443,
					},
				},
			},
			Selector: map[string]string{
				WebhookServerLabelKey: WebhookServerLabelValuePod,
			},
		},
	}
}

func GenerateServiceAccount(resource *installerv1alpha1.EKSPodIdentityWebhook) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ServiceAccountName,
			Namespace: Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(resource, schema.GroupVersionKind{
					Group:   installerv1alpha1.GroupVersion.Group,
					Version: installerv1alpha1.GroupVersion.Version,
					Kind:    "EKSPodIdentityWebhook",
				}),
			},
		},
	}
}

func GenerateRole(resource *installerv1alpha1.EKSPodIdentityWebhook) *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ServiceAccountName,
			Namespace: Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(resource, schema.GroupVersionKind{
					Group:   installerv1alpha1.GroupVersion.Group,
					Version: installerv1alpha1.GroupVersion.Version,
					Kind:    "EKSPodIdentityWebhook",
				}),
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				Verbs:     []string{"create"},
				APIGroups: []string{""},
				Resources: []string{"secrets"},
			},
			{
				Verbs:         []string{"get", "update", "patch"},
				APIGroups:     []string{""},
				Resources:     []string{"secrets"},
				ResourceNames: []string{SecretName},
			},
		},
	}
}

func GenerateRoleBinding(resource *installerv1alpha1.EKSPodIdentityWebhook, role *rbacv1.Role, sa *corev1.ServiceAccount) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ServiceAccountName,
			Namespace: Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(resource, schema.GroupVersionKind{
					Group:   installerv1alpha1.GroupVersion.Group,
					Version: installerv1alpha1.GroupVersion.Version,
					Kind:    "EKSPodIdentityWebhook",
				}),
			},
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      sa.Name,
				Namespace: sa.Namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     role.Name,
		},
	}
}

func GenerateClusterRole(resource *installerv1alpha1.EKSPodIdentityWebhook) *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: ServiceAccountName,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(resource, schema.GroupVersionKind{
					Group:   installerv1alpha1.GroupVersion.Group,
					Version: installerv1alpha1.GroupVersion.Version,
					Kind:    "EKSPodIdentityWebhook",
				}),
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				Verbs:     []string{"get", "watch", "list"},
				APIGroups: []string{""},
				Resources: []string{"serviceaccounts"},
			},
			{
				Verbs:     []string{"create", "get", "list", "watch"},
				APIGroups: []string{"certificates.k8s.io"},
				Resources: []string{"certificatesigningrequests"},
			},
		},
	}
}

func GenerateClusterRoleBinding(resource *installerv1alpha1.EKSPodIdentityWebhook, clusterRole *rbacv1.ClusterRole, sa *corev1.ServiceAccount) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: ServiceAccountName,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(resource, schema.GroupVersionKind{
					Group:   installerv1alpha1.GroupVersion.Group,
					Version: installerv1alpha1.GroupVersion.Version,
					Kind:    "EKSPodIdentityWebhook",
				}),
			},
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      sa.Name,
				Namespace: sa.Namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     clusterRole.Name,
		},
	}
}

func GenerateDaemonset(resource *installerv1alpha1.EKSPodIdentityWebhook) *appsv1.DaemonSet {
	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      DaemonsetName,
			Namespace: Namespace,
			Labels: map[string]string{
				WebhookServerLabelKey: "eks-webhook-daemonset",
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(resource, schema.GroupVersionKind{
					Group:   installerv1alpha1.GroupVersion.Group,
					Version: installerv1alpha1.GroupVersion.Version,
					Kind:    "EKSPodIdentityWebhook",
				}),
			},
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					WebhookServerLabelKey: WebhookServerLabelValuePod,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						WebhookServerLabelKey: WebhookServerLabelValuePod,
					},
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "webhook-certs",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:            DaemonsetName,
							Image:           "amazon/amazon-eks-pod-identity-webhook:latest",
							ImagePullPolicy: corev1.PullAlways,
							Command: []string{
								"/webhook",
								"--in-cluster",
								"--namespace=" + resource.Spec.Namespace,
								"--service-name=" + ServiceName,
								"--tls-secret=" + SecretName,
								"--annotation-prefix=eks.amazonaws.com",
								"--token-audience=" + resource.Spec.TokenAudience,
								"--logtostderr",
								"--v=4",
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "webhook-certs",
									ReadOnly:  false,
									MountPath: "/var/run/app/certs",
								},
							},
						},
					},
					ServiceAccountName: ServiceAccountName,
				},
			},
		},
	}
}
