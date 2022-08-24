# secret-mangler-operator

SecretMangler is a Kubernetes operator used to create secrets based on parts of other secrets or to mirror existing secrets.

## CRD explaination

An example for a secret constructed from parts of other secrets would look like the following:

```
---
apiVersion: secret-mangler.wreiner.at/v1alpha1
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
    cascadeMode: [KeepNoAction|KeepLostSync|RemoveLostSync|CascadeDelete]
    mappings:
      fixedmapping: "some-value-which-will-used-as-is"
      dynamicmapping: "[NAMESPACE/]OBJECT_NAME:LOOKUP_FIELD"
```

The _dynamicmapping_ field explained:

```
<[NAMESPACE/]OBJECT_NAME:LOOKUP_FIELD>
[NAMESPACE/] 	..  namespace of the referenced secret
                  If omitted the namespace of the SecretMangler object is used.
OBJECT_NAME   ..  name of the referenced secret
LOOKUP_FIELD  ..  key value of the Data field of the referenced secret
```

Please note: The SecretMangler object needs to be added in the same namespace as the secret it should generate.

### Edge Cases

There are different [edge cases](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#object-references) which need to be taken care of or at least be discussed when working with objects accross multiple namespaces.

The SecretMangler operator handles those edge cases with an _cascadeMode_ field:

```
cascadeMode: [KeepNoAction|KeepLostSync|RemoveLostSync|CascadeDelete]
```

On initial secret creation the secret is only created if _all_ dynamic field mappings are found. If not, the secret will not be initially created.

| cascadeMode    | Function                                                                                                                                                                                                                                        |
|----------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| KeepNoAction   | Keeps the secret the way it was initially created or last changed and no sync of changes in referenced secrets is performed.                                                                                                                                    |
| KeepLostSync   | Tries to sync data from referenced secrets.<br>If one or more sources are lost their data is kept as it was synced last.                                                                                                                        |
| RemoveLostSync | Tries to sync data from referenced secrets.<br>If one or more sources are lost their data will be removed from the created secret.<br>If no more sources are available and no fixed mappings are present the secret will be removed as a whole. |
| CascadeDelete  | Removes the secret entirely if only one source is lost no matter whether other sources are still present or not.                                                                                                                                |

For all modes if the data part of the secret would be empty the secret is being removed entirely.
#### Workflow

* Initial secret creation
  * [X] If not all dynamic mappings are found do not create
  * [X] Create the new secret if all dynamic mappings are found
* If the new secret was created earlier and a reference gets changed handle it with:
  * [X] KeepNoAction = keep as is - keep the new secret the way it was initially created and do not sync changes of sources
  * [X] KeepLostSync = keep lost but sync present - if one source was deleted keep existing data but update all sources which can be found
  * [X] RemoveLostSync = remove lost and sync present - if one source was delted remove its data and sync all other sources
    * [X] if no more sources and no fixedmapping is present delete the secret
  * [X] CascadeDelete = cascade delete - if one source was deleted remove the complete generated secret

## ToDo

* [X] subscribe to created secret to handle
* [X] subscribe to source secret
* [X] define edge cases
* [ ] implement mirror function

### mirror

An example for mirroring a secret:

```
---
apiVersion: secret-mangler.wreiner.at/v1alpha1
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
#### mirror vs mappings

Please note that _mirror_ and _mappings_ will be mutually exclusive.

If mirror is defined the referenced secret data is mirrored as a whole and also the secret type is kept from the referenced secret. Labels and annotations from the referenced secret are not kept. Mappings are not possible in this situation.

If mappings is defined mirror can not be defined too.

## Sources

* [CODE4104: Let's build a Kubernetes Operator in Go! with Michael Gasch & Rafael Brito](https://www.youtube.com/watch?v=8Ex7ybi273g)
  * [Example Source Code](https://github.com/embano1/codeconnect-vm-operator)
