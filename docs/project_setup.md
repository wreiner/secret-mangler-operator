# How secret-mangler was setup using kubebuilder

```
kubebuilder init --domain secret-mangler.wreiner.at --repo github.com/wreiner/secret-mangler-operator
kubebuilder create api --version v1alpha1 --kind SecretMangler
```

```
Next: implement your new API and generate the manifests (e.g. CRDs,CRs) with:
$ make manifests
```