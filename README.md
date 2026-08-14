# Provider Dynatrace IAM

`provider-dynatrace-iam` is a [Crossplane](https://crossplane.io/) provider that is built using [Upjet](https://github.com/crossplane/upjet) code generation tools and exposes XRM-conformant managed resources for the Dynatrace IAM API.

## Features

- **IAM Resources**: IAM Groups, Policies, Policy Bindings (v1 & v2), Policy Boundaries, Permissions, Service Users, and Users.
- **Management Zone Permissions**: `Permission` in `mgmz.dynatrace.crossplane.io`.
- **Policy & Users**: Policy bindings and Users in `policy.dynatrace.crossplane.io` and `user.dynatrace.crossplane.io`.
- **Custom CostCenter Controller**: Direct account-level Cost Center management.
- **Both Cluster-Scoped and Namespaced APIs** supported.
- **SafeStart enabled**: Dynamically gates controllers based on CRDs present in the cluster.

## Getting Started & Testing

Follow the comprehensive testing and local development guides:
- [Local Release Guide](local_release.md)
- [How to Test Guide](how_to_test.md)

## Developing

Run code-generation pipeline:
```console
export PATH=$PATH:$HOME/go/bin
unset GOROOT
export GOTOOLCHAIN=local
make generate
```

Run against a local Kubernetes cluster:
```console
export PATH=$PATH:$HOME/go/bin
unset GOROOT
export GOTOOLCHAIN=local
make run
```

Build local package:
```console
./scripts/build_local.sh -v v0.1.0 -p linux_amd64
```
