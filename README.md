# Provider Dynatrace IAM

`provider-dynatrace-iam` is a [Crossplane](https://crossplane.io/) provider that is built using [Upjet](https://github.com/crossplane/upjet) code generation tools and exposes XRM-conformant managed resources for the Dynatrace IAM API.

## Features

- **IAM Resources**: IAM Groups, Policies, Policy Bindings (v1 & v2), Policy Boundaries, Permissions, Service Users, and Users.
- **Management Zone Permissions**: `Permission` in `mgmz.dynatrace.crossplane.io`.
- **Policy & Users**: Policy bindings and Users in `policy.dynatrace.crossplane.io` and `user.dynatrace.crossplane.io`.
- **Custom CostCenter Controller**: Direct account-level Cost Center management.
- **Both Cluster-Scoped and Namespaced APIs** supported.
- **SafeStart enabled**: Dynamically gates controllers based on CRDs present in the cluster.

## Developing

Run code-generation pipeline:
```console
export PATH=$PATH:$HOME/go/bin
unset GOROOT
export GOTOOLCHAIN=local
make generate
```
