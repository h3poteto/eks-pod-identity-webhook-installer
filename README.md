# EKS Pod Identity Webhook Installer
This is a controller to install [Amazon EKS Pod Identity Webhook](https://github.com/aws/amazon-eks-pod-identity-webhook) to your Kubernetes cluster.

## Overview
When you are building Kubernetes clusters on AWS by a method other than EKS, you have to install eks-pod-identity-webhook to use IAM Role For Service Account (IRSA). The official repository provides [Makefile](https://github.com/aws/amazon-eks-pod-identity-webhook/blob/master/Makefile). But sometimes you have to rewrite parameters of the deploymente before make command, because we use other audience and issuer for bare metal clusters.
This controller can automatically install its webhook server without make command. Therefore this repository provides another way to install eks-pod-identity-webhook in your cluster.

## How to use it
Please apply custom resource, for example:

```yaml
apiVersion: installer.h3poteto.dev/v1alpha1
kind: EKSPodIdentityWebhook
metadata:
  name: my-example
spec:
  issuerHost: "amazonaws.com"
  namespace: "default"
```

After that, pod-identity-webhook pods are deployed, and CertificateSigningRequests are approved.

## How to install
TODO

## License
The software is available as open source under the terms of the [Apache License 2.0](https://www.apache.org/licenses/LICENSE-2.0).
