# secret-mangler

SecretMangler is a Kubernetes operator used to create secrets based on parts of other secrets or to mirror existing secrets.

## CRD explaination

An example for a secret constructed from parts of other secrets would look like the following:

```
---
apiVersion: secret-mangler.wreiner.at.secret-mangler.wreiner.at/v1alpha1
kind: SecretMangler
metadata:
  name: mangler01
  namespace: aha
spec:
  secretTemplate:
    apiVersion: v1
    kind: Secret
    namespace: aha
    name: "mangler01-secret"
    mappings:
      fixedmapping: "some-value-which-will-used-as-is"
      dynamicmapping: "[NAMESPACE/]OBJECT_NAME:LOOKUP_FIELD"
```

The _dynamicmapping_ field explained:

```
<[NAMESPACE/]OBJECT_NAME:LOOKUP_FIELD>
[NAMESPACE/] 	..  namespace of the referenced secret
                  if omitted the current current namespace is used
                    connecting to other namespaces should be avoided due to edge cases
                    if not avoided edge case behaviour needs to be clearly documented
                    (see further down about edge cases)
OBJECT_NAME   ..  the name of the referenced secret
LOOKUP_FIELD  ..  the key value of the Data field of the referenced secret
```

An example for mirroring a secret:

```
---
apiVersion: secret-mangler.wreiner.at.secret-mangler.wreiner.at/v1alpha1
kind: SecretMangler
metadata:
  name: mangler01
  namespace: aha
spec:
  secretTemplate:
    apiVersion: v1
    kind: Secret
    namespace: aha
    name: "mangler01-secret"
    mirror: <[NAMESPACE/]OBJECT_NAME:LOOKUP_FIELD>
```

Please note: The SecretMangler object needs to be added in the same namespace as the secret it should generate.

### mirror vs mappings

Please note that _mirror_ and _mappings_ are mutually exclusive.

If mirror is defined the referenced secret data is mirrored as a whole and also the secret type is kept from the referenced secret. Labels and annotations from the referenced secret are not kept. Mappings are not possible in this situation.

If mappings is defined mirror can not be defined too.

### Edge Cases

It is yet to be determined how to handle different [edge cases](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#object-references).

An idea is to add a _cascadeMode_ field like this:

```
cascadeMode: [KeepNoAction|KeepLostSync|RemoveLostSync|CascadeDelete]
```

#### Workflow

* Initial secret creation
  * [ ] If not all dynamic mappings are found do not create
  * [X] Create the new secret if all dynamic mappings are found
* If the new secret was created earlier and a reference gets changed handle it with:
  * [X] KeepNoAction = keep as is - keep the new secret the way it was initially created and do not sync changes of sources
  * [ ] KeepLostSync = keep lost but sync present - if one source was deleted keep existing data but update all sources which can be found
  * [ ] RemoveLostSync = remove lost and sync present - if one source was delted remove its data and sync all other sources
    * if no more sources and no fixeddata is present delete the secret
  * [ ] CascadeDelete = cascade delete - if one source was deleted remove the complete generated secret

## ToDo

* [ ] subscribe to created secret to handle
* [X] subscribe to source secret
* [ ] implement mirror function
* [X] define edge cases

## Sources

* [CODE4104: Let's build a Kubernetes Operator in Go! with Michael Gasch & Rafael Brito](https://www.youtube.com/watch?v=8Ex7ybi273g)
  * [Example Source Code](https://github.com/embano1/codeconnect-vm-operator)