/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"strings"

	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler" // Required for Watching
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"gitea.wreiner.at/wreiner/secret-mangler/api/v1alpha1"
	secretmanglerwreineratv1alpha1 "gitea.wreiner.at/wreiner/secret-mangler/api/v1alpha1"
)

// SecretManglerReconciler reconciles a SecretMangler object
type SecretManglerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=secret-mangler.wreiner.at.secret-mangler.wreiner.at,resources=secretmanglers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=secret-mangler.wreiner.at.secret-mangler.wreiner.at,resources=secretmanglers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=secret-mangler.wreiner.at.secret-mangler.wreiner.at,resources=secretmanglers/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=secrets/status,verbs=get

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the SecretMangler object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile

func (r *SecretManglerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	var secretMangler v1alpha1.SecretMangler
	if err := r.Get(ctx, req.NamespacedName, &secretMangler); err != nil {
		log.Error(err, "unable to fetch SecretMangler")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	msg := fmt.Sprintf("received reconcile request for %q (namespace: %q)", secretMangler.GetName(), secretMangler.GetNamespace())
	log.Info(msg)

	existingSecret := RetrieveSecret(secretMangler.Spec.SecretTemplate.Name, secretMangler.Spec.SecretTemplate.Namespace, r, ctx)
	if existingSecret == nil {
		// create secret on the cluster
		log.Info("did not find existing secret, will try to create new secret ..")

	// build the secret
		newSecret := SecretBuilder(&secretMangler, nil, r, ctx)
	if newSecret == nil {
			msg = fmt.Sprintf("building the secret failed for reconcile request [%q/%q]", secretMangler.GetNamespace(), secretMangler.GetName())
			log.Info(msg)
		return ctrl.Result{}, nil
	}
	log.Info("after builder")

		msg = fmt.Sprintf("will create secret for reconcile request [%q/%q]", secretMangler.GetNamespace(), secretMangler.GetName())
		log.Info(msg)

		if err := r.Create(ctx, newSecret); err != nil {
			log.Error(err, "unable to create secret for SecretMangler")
			return ctrl.Result{}, err
		}
	} else {
		// work on a previously created secret
		log.Info("found existing secret, will check fields ..")

		// with KeepNoAction the existing secret which was created on an earlier run will be kept as is
		if secretMangler.Spec.SecretTemplate.CascadeMode == "KeepNoAction" {
			msg = fmt.Sprintf("will not attempt sync because cascademode KeepNoAction for reconcile request %q (namespace: %q)", secretMangler.GetName(), secretMangler.GetNamespace())
			log.Info(msg)
			return ctrl.Result{}, nil
		}

		// get updated secret data
		newData := make(map[string][]byte)
		error := DataBuilder(&secretMangler, &newData, false, r, ctx)
		if error == false {
			msg = fmt.Sprintf("building secret data failed for reconcile request [%q/%q]", secretMangler.GetNamespace(), secretMangler.GetName())
			log.Info(msg)
			return ctrl.Result{}, nil
			}

		actionIndicator := CompareExistingSecretDataToNewData(&secretMangler, &existingSecret.Data, &newData, ctx)
		switch actionIndicator {
		case 0:
			// nothing todo
			msg = fmt.Sprintf("secret data has not changed for reconcile request [%q/%q]", secretMangler.GetNamespace(), secretMangler.GetName())
					log.Info(msg)
			return ctrl.Result{}, nil
		case 1:
			// update needed
			msg = fmt.Sprintf("secret data has changed, will update for reconcile request [%q/%q]", secretMangler.GetNamespace(), secretMangler.GetName())
					log.Info(msg)

			// build the secret
			newSecret := SecretBuilder(&secretMangler, &newData, r, ctx)
			if newSecret == nil {
				msg = fmt.Sprintf("building the secret failed for reconcile request [%q/%q]", secretMangler.GetNamespace(), secretMangler.GetName())
					log.Info(msg)
				return ctrl.Result{}, nil
		}
			log.Info("after builder")

			if err := r.Update(ctx, newSecret); err != nil {
				log.Error(err, "unable to update secret for for reconcile request [%q/%q]", secretMangler.GetNamespace(), secretMangler.GetName())
				return ctrl.Result{}, err
	}
		case 2:
			// delete needed
			msg = fmt.Sprintf("secret will be deleted for reconcile request [%q/%q]", secretMangler.GetNamespace(), secretMangler.GetName())
			log.Info(msg)

			if err := r.Delete(ctx, existingSecret); err != nil {
				log.Error(err, "unable to delete secret for for reconcile request [%q/%q]", secretMangler.GetNamespace(), secretMangler.GetName())
				return ctrl.Result{}, err
		}
		}
	}

	msg = fmt.Sprintf("secret was worked on, now status for reconcile request %q (namespace: %q)", secretMangler.GetName(), secretMangler.GetNamespace())
	log.Info(msg)

	// FIXME set useful status
	// set status to true
	secretMangler.Status.SecretCreated = true

	// .. and update the status
	if err := r.Status().Update(ctx, &secretMangler); err != nil {
		log.Error(err, "unable to update SecretMangler status")
		return ctrl.Result{}, err
	}

	fmt.Println("status updated")

	return ctrl.Result{}, nil
}

// IsLookupString checks if a string starts with < and ends with > which indicates a lookup string.
func IsLookupString(lookupString string) (isLookupString bool) {
	if strings.HasPrefix(lookupString, "<") && strings.HasSuffix(lookupString, ">") {
		return true
	}
	return false
}

// ParseLookupString will parse a lookupString used in mappings or mirror.
// If no namespace was given an empty string will be returned instead of a namespace.
// If the lookupString does not at least contain a secret and a field reference false will be returned for ok.
func ParseLookupString(lookupString string) (namespaceName string, existingSecretName string, existingSecretField string, ok bool) {
	var newFieldValue string

	// split by / indicates a provided namespace of the secret to lookup
	splitArray := strings.Split(lookupString, "/")
	if len(splitArray) > 1 {
		// remove now unneeded characters
		namespaceName = strings.TrimLeft(splitArray[0], "<")
		newFieldValue = strings.TrimRight(splitArray[1], ">")
	}

	// split by : delimits the scret name and the lookup field in the secret
	splitArray = strings.Split(newFieldValue, ":")
	if len(splitArray) > 1 {
		existingSecretName = splitArray[0]
		existingSecretField = splitArray[1]

		// set true only if all information could be gathered
		ok = true
	}

	return namespaceName, existingSecretName, existingSecretField, ok
}

// CompareExistingSecretDataToNewData compares to data maps of Secrets.
// It will return 0 on equal, 1 on Secret needs update, 2 on Secret needs to be deleted
func CompareExistingSecretDataToNewData(secretManglerObject *v1alpha1.SecretMangler, existingSecretData *map[string][]byte, newData *map[string][]byte, ctx context.Context) int {
	log := log.FromContext(ctx)
	logMsg := ""
	needUpdate := false

	for checkKey, checkValue := range *existingSecretData {
		fmt.Printf("got [%s: %b] ..\n", checkKey, checkValue)

		// https://stackoverflow.com/a/36463704
		if val, ok := (*newData)[checkKey]; ok {
			// when the key is found in the new map it is already newest
			// so nothing is todo in this case so we can continue on
			fmt.Printf("found key [%s: %b] in newData\n", checkKey, val)
			continue
		}

		if secretManglerObject.Spec.SecretTemplate.CascadeMode == "KeepLostSync" {
			logMsg = fmt.Sprintf("keeping key %s because of KeepLostSync for reconcile request [%q/%q]",
				checkKey, secretManglerObject.GetNamespace(), secretManglerObject.GetName())
			log.Info(logMsg)

			// keep old data which was lost in this reconcile run
			(*newData)[checkKey] = checkValue
			needUpdate = true

		} else if secretManglerObject.Spec.SecretTemplate.CascadeMode == "RemoveLostSync" {
			// just log the message
			logMsg = fmt.Sprintf("removing key %s from data because of RemoveLostSync for reconcile request [%q/%q]",
				checkKey, secretManglerObject.GetNamespace(), secretManglerObject.GetName())
			log.Info(logMsg)

		} else if secretManglerObject.Spec.SecretTemplate.CascadeMode == "CascadeDelete" {
			logMsg = fmt.Sprintf("removing complete secret because of CascadeDelete for reconcile request [%q/%q]",
				secretManglerObject.GetNamespace(), secretManglerObject.GetName())
			log.Info(logMsg)

			// secret should be deleted
			return 2
		}
	}

	// sanity check - if newData is empty delete the secret as there is no more data to store in the secret
	if len(*newData) == 0 {
		logMsg = fmt.Sprintf("removing complete secret because there is no data to store for reconcile request [%q/%q]",
			secretManglerObject.GetNamespace(), secretManglerObject.GetName())
		log.Info(logMsg)
		return 2
	}

	if needUpdate == true {
		return 1
	}

	return 0
}

// RetrieveSecret retrieves a secret from the Kubernetes cluster with a given Name and Namespace.
func RetrieveSecret(existingSecretName, namespaceName string, r *SecretManglerReconciler, ctx context.Context) *v1.Secret {
	log := log.FromContext(ctx)

	var existingSecret v1.Secret

	namespacedNameExistingSecret := types.NamespacedName{Namespace: namespaceName, Name: existingSecretName}

	if err := r.Get(ctx, namespacedNameExistingSecret, &existingSecret); err != nil {
		logMsg := fmt.Sprintf("unable to fetch secret %s/%s - %s", namespaceName, existingSecretName, err.Error())
		log.Info(logMsg)
		return nil
	}

	return &existingSecret
}

// DataBuilder generates the data mappings of a secret from a SecretMangler object.
func DataBuilder(secretManglerObject *v1alpha1.SecretMangler, newData *map[string][]byte, returnOnSourceNotFound bool, r *SecretManglerReconciler, ctx context.Context) bool {
	log := log.FromContext(ctx)

	if newData == nil {
		logMsg := "provided newdata map is nil in DataBuilder, data cannot be build .."
		log.Info(logMsg)
		return false
	}

	for newField, newFieldValue := range secretManglerObject.Spec.SecretTemplate.Mappings {
		fmt.Println("newField:", newField, "newFieldValue:", newFieldValue)

		// check if value should be treated as a lookupString
		if IsLookupString(newFieldValue) {
			fmt.Printf("value of field %s indicates a dynamic field\n", newField)

			namespaceName, existingSecretName, existingSecretField, ok := ParseLookupString(newFieldValue)
			if ok == false {
				logMsg := fmt.Sprintf("dynamic mapping %s contains a faulty lookup string %s", newField, newFieldValue)
				// FIXME log correctly
				// log.Error(logMsg)
				log.Info(logMsg)
				return false
			}

			// use the namespace of the CR if no explicit namespace is set to lookup existing secret
			if namespaceName == "" {
				namespaceName = secretManglerObject.Namespace
			}

			// fetch secret
			existingSecret := RetrieveSecret(existingSecretName, namespaceName, r, ctx)
			if existingSecret == nil {
				if returnOnSourceNotFound {
					return false
			}
				continue
			}

			// https://stackoverflow.com/a/2050629
			if existingSecretFieldValue, found := existingSecret.Data[existingSecretField]; found {
				fmt.Printf("will add %s: %s to newData ..\n", newField, existingSecretField)
				(*newData)[newField] = existingSecretFieldValue
			}
		} else {
			fmt.Printf("will add %s: %s to newData ..\n", newField, newFieldValue)

			// fixed value can be added as is
			(*newData)[newField] = []byte(newFieldValue)
		}

		fmt.Println("----")
	}

	return true
}

// SecretBuilder generates a secret based on a SecretMangler object with all data and metadata.
// The secret will not be applied to the Kubernetes cluster.
func SecretBuilder(secretManglerObject *v1alpha1.SecretMangler, givenData *map[string][]byte, r *SecretManglerReconciler, ctx context.Context) *v1.Secret {
	log := log.FromContext(ctx)

	// Build the data mappings of the secret if it is not given
	newData := make(map[string][]byte)
	if givenData == nil || len((*givenData)) == 0 {
		log.Info("no data or emtpy data given to SecretBuilder, trying to obtain data ..")
		ok := DataBuilder(secretManglerObject, &newData, true, r, ctx)
		if ok == false {
			log.Info("cannot obtain data, cannot go on ..")
			return nil
	}
	} else {
		newData = *givenData
	}

	// Build the whole secret
	newSecret := &v1.Secret{
		ObjectMeta: v12.ObjectMeta{
			Name:      secretManglerObject.Spec.SecretTemplate.Name,
			Namespace: secretManglerObject.Spec.SecretTemplate.Namespace,
			// Labels: secretManglerObject.Spec.SecretTemplate.Label,*
		},
		Data: newData,
		Type: "Opaque",
	}

	// Set the owner reference.
	// This allows the Kubernetes garbage collector
	// to clean up secrets when we delete the SecretMangler, and allows controller-runtime to figure out
	// which SecretMangler needs to be reconciled when a given secret changes (is added, deleted, completes, etc).
	if err := ctrl.SetControllerReference(secretManglerObject, newSecret, r.Scheme); err != nil {
		log.Error(err, "error in setting owner reference to secret")
		return nil
	}

	return newSecret
}

func OldSecretBuilder(cr *v1alpha1.SecretMangler) *v1.Secret {
	return &v1.Secret{
		ObjectMeta: v12.ObjectMeta{
			Name:      cr.Spec.SecretTemplate.Name,
			Namespace: cr.Spec.SecretTemplate.Namespace,
			// Labels: cr.Spec.SecretTemplate.Label,*
		},
		// Data: map[string][]byte{
		// 	AdminUsernameProperty: []byte("admin"),
		// 	AdminPasswordProperty: []byte(GenerateRandomString(10)),
		// },
		Data: map[string][]byte{
			"test": []byte("ZGVydGVzdGRlcg=="),
		},
		// Data: cr.Spec.SecretTemplate.Mappings,
		Type: "Opaque",
	}
}

// secret := &v1.Secret{
// 	ObjectMeta: v1.ObjectMeta{
// 		Name: tls.SecretName,
// 	},
// 	Data: map[string][]byte{
// 		v1.TLSCertKey:       cert,
// 		v1.TLSPrivateKeyKey: key,
// 	},
// }

// SetupWithManager sets up the controller with the Manager.
func (r *SecretManglerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// set up a real clock, since we're not in a test
	// if r.Clock == nil {
	// 	r.Clock = realClock{}
	// }

	// if err := mgr.GetFieldIndexer().IndexField(context.Background(), &kbatch.Job{}, jobOwnerKey, func(rawObj client.Object) []string {
	// 	// grab the job object, extract the owner...
	// 	job := rawObj.(*kbatch.Job)
	// 	owner := metav1.GetControllerOf(job)
	// 	if owner == nil {
	// 		return nil
	// 	}
	// 	// ...make sure it's a CronJob...
	// 	if owner.APIVersion != apiGVStr || owner.Kind != "CronJob" {
	// 		return nil
	// 	}

	// 	// ...and if so, return it
	// 	return []string{owner.Name}
	// }); err != nil {
	// 	return err
	// }

	return ctrl.NewControllerManagedBy(mgr).
		For(&secretmanglerwreineratv1alpha1.SecretMangler{}).
		Owns(&v1.Secret{}).
		Watches(
			&source.Kind{Type: &v1.Secret{}},
			handler.EnqueueRequestsFromMapFunc(func(obj client.Object) []reconcile.Request {
				fmt.Println("in watches function")
				secret, ok := obj.(*v1.Secret)
				if !ok {
					// FIXME
					fmt.Println("secret is nil?")
					return nil
				}

				fmt.Printf("Secret [%s/%s] changed, checking for corresponding SecretMangler object ..\n", secret.Namespace, secret.Name)

				var reconcileRequests []reconcile.Request
				secretManglerList := &secretmanglerwreineratv1alpha1.SecretManglerList{}
				client := mgr.GetClient()

				err := client.List(context.TODO(), secretManglerList)
				if err != nil {
					return []reconcile.Request{}
				}

				fmt.Println("will iterate over SecretMangler objects and their mappings/mirrors ..")
				for _, secretManglerObj := range secretManglerList.Items {
					fmt.Printf("got SecretMangler object [%s/%s]\n", secretManglerObj.Namespace, secretManglerObj.Name)

					// FIXME needed when mirror is implemented
					// if secretManglerObj.Spec.SecretTemplate.Mirror != "" {

					// }
					if len(secretManglerObj.Spec.SecretTemplate.Mappings) != 0 {
						for field, fieldValue := range secretManglerObj.Spec.SecretTemplate.Mappings {
							if IsLookupString(fieldValue) {
								// get namespace and name from the dynamic field
								// if current secret is part of the dynamic field add to reconciliation request
								fmt.Printf("value [%s] of field %s indicates a dynamic field\n", fieldValue, field)

								referencedSecretNamespaceName, referencedSecretName, _, ok := ParseLookupString(fieldValue)
								if ok == false {
									logMsg := fmt.Sprintf("dynamic mapping %s contains a faulty lookup string %s", field, fieldValue)
									// FIXME log correctly
									// log.Error(logMsg)
									fmt.Println(logMsg)
									return nil
								}

								// check if secretNames match
								if secret.Name == referencedSecretName {
									// if no explicit namesapce is given in the mapping the namespace of the SecretMangler object is used
									if referencedSecretNamespaceName == "" {
										// i think it's wrong
										// referencedSecretNamespaceName = secretManglerObj.Spec.SecretTemplate.Namespace
										referencedSecretNamespaceName = secretManglerObj.Namespace
									}

									fmt.Printf("found reference to [%s/%s]\n", referencedSecretNamespaceName, referencedSecretName)

									// check if the secret is in the same namespace
									if referencedSecretNamespaceName == secret.Namespace {
										fmt.Printf("will add SecretMangler object [%s/%s] to reconciliation requests ..\n", secretManglerObj.Namespace, secretManglerObj.Name)
										// append secretMangler to reconcileRequests
										reconcileRequests = append(reconcileRequests, reconcile.Request{
											NamespacedName: types.NamespacedName{
												Name:      secretManglerObj.Name,
												Namespace: secretManglerObj.Namespace,
											},
										})

										// we can break now and check next SecretMangler object
										break
									}
								}
							}
						}
					}
				}

				fmt.Println("will leave watches function")
				fmt.Println("------ watch ------")
				return reconcileRequests
			}),
		).
		Complete(r)
}
