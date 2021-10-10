# EKS Pod Identity Webhook Installer
This is a controller to install [Amazon EKS Pod Identity Webhook](https://github.com/aws/amazon-eks-pod-identity-webhook) to your Kubernetes cluster.

## Overview
When you are building Kubernetes clusters on AWS by a method other than EKS, you have to install eks-pod-identity-webhook to use IAM Role For Service Account (IRSA). The official repository provides [Makefile](https://github.com/aws/amazon-eks-pod-identity-webhook/blob/master/Makefile). But sometimes you have to rewrite parameters of the deploymente before make command, because we use other audience and issuer for bare metal clusters.
This controller can automatically install its webhook server without make command. Therefore this repository provides another way to install eks-pod-identity-webhook in your cluster.

## How to install
You can install this controller using Helm.

```
$ helm repo add h3poteto-stable https://h3poteto.github.io/charts/stable
$ helm install my-installer --namespace kube-system h3poteto-stable/eks-pod-identity-webhook-installer
```

Please refer [helm repository](https://github.com/h3poteto/charts/tree/master/stable/eks-pod-identity-webhook-installer) for parameters.

## How to use it
You can customize `tokenAudience` and `namespace` which are applied for eks-pod-identity-webhook.
Please change `tokenAudience` according to your audience. And eks-pod-identity-webhook pod runs in `namespace`.

For example,

```
$ helm install my-installer --namespace kube-system \
  --set eksPodIdentityWebhookInstaller.tokenAudience=amazonaws.com \
  --set eksPodIdentityWebhookInstaller.namespace=default
```

After that, pod-identity-webhook pods are deployed in default namespace, and CertificateSigningRequests are approved.


## License
The software is available as open source under the terms of the [Apache License 2.0](https://www.apache.org/licenses/LICENSE-2.0).
