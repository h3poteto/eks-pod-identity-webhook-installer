// Refs: https://github.com/aws/amazon-eks-pod-identity-webhook/tree/master/deploy

package controllers

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"time"

	installerv1alpha1 "github.com/h3poteto/eks-pod-identity-webhook-installer/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	serverKeyName           = "key.pem"
	serverCertName          = "cert.pem"
	WebhookServerLabelKey   = "ekspodidentitywebhooks.installer.h3poteto.dev"
	WebhookServerLabelValue = "eks-webhook"
)

func generateSecret(resource *installerv1alpha1.EKSPodIdentityWebhook, namespace, name string, key, cert []byte) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				WebhookServerLabelKey: WebhookServerLabelValue,
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(resource, schema.GroupVersionKind{
					Group:   installerv1alpha1.GroupVersion.Group,
					Version: installerv1alpha1.GroupVersion.Version,
					Kind:    "EKSPodIdentityWebhook",
				}),
			},
		},
		Data: map[string][]byte{
			serverKeyName:  key,
			serverCertName: cert,
		},
		Type: corev1.SecretTypeOpaque,
	}
}

func newCertificates(serviceName, namespace string) ([]byte, []byte, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	name := pkix.Name{
		Country:      []string{},
		Organization: []string{},
		Locality:     []string{},
		CommonName:   serviceName + "." + namespace + ".svc",
	}

	CA := x509.Certificate{
		SerialNumber:          big.NewInt(2048),
		Subject:               name,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		// To generate extensions included SANs.
		DNSNames: []string{
			serviceName + "." + namespace + ".svc",
		},
	}
	cert, err := x509.CreateCertificate(rand.Reader, &CA, &CA, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, nil, err
	}
	keyPem := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}
	certPem := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert,
	}

	return pem.EncodeToMemory(keyPem), pem.EncodeToMemory(certPem), nil
}
