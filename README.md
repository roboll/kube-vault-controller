# kube-vault-controller [![CircleCI](https://circleci.com/gh/roboll/kube-vault-controller.svg?style=svg)](https://circleci.com/gh/roboll/kube-vault-controller)

[![Docker Repository on Quay](https://quay.io/repository/roboll/kube-vault-controller/status "Docker Repository on Quay")](https://quay.io/repository/roboll/kube-vault-controller)

Claim secrets from [Vault](https://vaultproject.io) for use in Kubernetes.

## Features

* Provide secrets from Vault to applications in Kubernetes via claims.
* Use Kubernetes secret objects, including TLS type for ingress.
* Configurable lease renewal buffer, automatically rotate secrets for expiring leases.
* Easy ops: no persistent storage, everything stored in Kubernetes.
* [Namespaced secrets](#namespaced-secrets): Enforcing that secrets are only accessed per namespace


## TODO

* Per secret Vault authentication with [token](https://www.vaultproject.io/docs/auth/token.html) or [app role](https://www.vaultproject.io/docs/auth/approle.html).
* Add `--ingres-label` flag and watch ingress to fulfill tls spec.
* Template several secret values into a single datom.
* Add service account and RBAC role into chart.
* Support `time.Duration` for `renew`.
* Write user guide.

## Install

Install with [helm](https://github.com/kubernetes/helm): ([chart](./deploy/chart)), or kubectl: [templates](./deploy/chart/templates/).

## Usage

For more detailed usage see the [user guide](docs/user-guide.md).

Request secrets via `secretclaim`:

`kubectl create -f some-secret.yaml`

```
kind: SecretClaim
apiVersion: vaultproject.io/v1
metadata:
  name: some-secret
spec:
  type: Opaque
  path: secret/example
  renew: 3600
```

A secret by the same name, in the same namespace, will be created:

`kubectl get secret some-secret -o yaml`

```
kind: Secret
apiVersion: v1
data:
  field_one: base64-encoded-value
  field_two: base64-encoded-value
type: Opaque
metadata:
  name: some-secret
  namespace: kube-system
  annotations:
    vaultproject.io/lease-expiration: "1477272978"
    vaultproject.io/lease-id: "vault-lease-id"
    vaultproject.io/renewable: "false"
```

Reference the secret normally:

`kubectl create -f secret-consumer.yaml`

```
kind: Pod
apiVersion: v1
metadata:
  name: secret-consumer
spec:
  containers:
    - name: secret-consumer
      image: alpine:3.4
      command:
        - /bin/sh
        - -c
        - echo $SECRET_VALUE && cat /etc/secrets/field_one
      env:
        - name: SECRET_VALUE
          valueFrom:
            secretKeyRef:
              name: some-secret
              key: field_one
      volumeMounts:
        - name: secret-mount
          mountPath: /etc/secrets
  volumes:
    - name: secret-mount
      secret:
        secretName: some-secret
        items:
          - key: field_one
            path: field_one

```

## About

The controller is built with https://github.com/kubernetes/client-go, specifically the [`Informer`](https://github.com/kubernetes/client-go/blob/c72e2838b9cfac95603049d57c9abba12e587fff/tools/cache/controller.go#L196) API which makes watching for resources changes quite simple. The controller is triggered by changes from streaming updates via watch, and also syncs all resources each `sync-period`. The sync period is critical as it ensures all resources are examined periodically, allowing the application to remain stateless and not schedule operations in advance - when a secret is examined and the lease expiration is within it's claimed renewal period, the lease is renewed (if renewable) or the secret is rotated. To ensure secrets are renewed before their lease expires, ensure your sync period is smaller than your smallest claimed renewal time.

## Namespaced secrets

This feature is useful if you are running a Kubernetes cluster as a service and want kube-vault-controller to namespace secrets access. 

By adding a `--namespace-prefix` to the arguments you can ensure that all secretClaims starting with this prefix will only be able to access their own namespace. Secrets starting with this prefix need to be written to `${prefix}${namespace}/${key}`. Secrets that fall outside this path will still be globally accessible for the cluster. 

### Example usage

* Create a vault policy for your cluster with the desired prefix
  ```
  path "secret/cluster-name/*" {
    capabilities = ["read"]
  }
  ```
* Run kube-vault-controller with this prefix
  ```
  --namespace-prefix='secret/cluster-name/'
  ```

Secretclaims created in the namespace `example` will now only be able to create secretClaims with paths inside `secret/cluster-name/example/*`. If they try to access another path under the prefix that isn't their namespace they will see the following error:

```
2019/11/26 10:52:09 error: failed to update secret for key example/not-allowed: vault-controller: "example/not-allowed": can't create path "secret/cluster-name/othernamespace/not-allowed" because it is under the namespacePrefix "secret/cluster-name/" but not in its own namespace "example"
```

The prefix can be anything you want it to be. If you want your secrets to be in the form `secret/cluster-name/namespace` you will need to make sure that your prefix is exactly `secret/cluster-name/` with a trailing `/`. You could also use `secret/cluster-name_` as your prefix. This would mean secrets for the `example` namespace need to be written to `secret/cluster-name_example/key`. 

You can also look at the [namespaced-secrets example](./example/namespaced-secrets.yaml) to get a better idea of how it works. 
