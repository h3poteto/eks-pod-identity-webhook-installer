module github.com/h3poteto/eks-pod-identity-webhook-installer

go 1.16

require (
	github.com/go-logr/logr v0.3.0
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.2
	k8s.io/api v0.20.2
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v0.20.2
	k8s.io/utils v0.0.0-20210820185131-d34e5cb4466e
	sigs.k8s.io/controller-runtime v0.8.3
)
