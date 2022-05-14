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
	"sigs.k8s.io/controller-runtime/pkg/log"

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

	// build the secret
	newsec := SecretBuilder(&secretMangler, r, ctx)
	if newsec == nil {
		return ctrl.Result{}, nil
	}

	// create secret on the cluster
	if err := r.Create(ctx, newsec); err != nil {
		log.Error(err, "unable to create secret for SecretMangler", secretMangler)
		return ctrl.Result{}, err
	}

	// // read an existing secret
	// var existingSecret v1.Secret
	// fooID := types.NamespacedName{Namespace: "gitea", Name: "gitea-admin-secret"}
	// if err := r.Get(ctx, fooID, &existingSecret); err != nil {
	// 	log.Error(err, "unable to fetch gitea admin secret")
	// }

	// fmt.Println("--- debug 0 ---")
	// fmt.Println(existingSecret.Data)
	// fmt.Println("--- debug 0 ---")

	// // r.Client.Get()

	// fmt.Println("--- debug 1 ---")
	// fmt.Println(secretMangler.Spec.SecretTemplate.Name)
	// fmt.Println("--- debug 1 ---")

	// fmt.Println("--- debug 2 ---")
	// fmt.Println("--- map iteration")
	// // https://stackoverflow.com/a/8018909
	// for k, v := range secretMangler.Spec.SecretTemplate.Mappings {
	// 	fmt.Println("k:", k, "v:", v)
	// }
	// fmt.Println("--- debug 2 ---")

	// secret := &api.Secret{
	// 	ObjectMeta: api.ObjectMeta{
	// 		Name: secretMangler.Spec.NewName,
	// 	},
	// 	Data: map[string][]byte{
	// 		keyName: buffer,
	// 	},
	// }

	// secret := SecretBuilder(&secretMangler)

	// log.V(1).Info("--- wreiner\n")
	// spew.Dump(ctx)
	// spew.Dump(secretMangler)
	// log.V(1).Info("--- wreiner\n")

	return ctrl.Result{}, nil
}

// Will parse a lookupString used in mappings.
// If no namespace was given an empty string will be returned instead of a namespace.
// If the lookupString does not at least contain a secret and a field reference false will be returned for ok.
func parseLookupString(lookupString string) (namespaceName string, existingSecretName string, existingSecretField string, ok bool) {
	var newFieldValue string

	// split by / indicates a provided namespace of the secret to lookup
	splitArray := strings.Split(lookupString, "/")
	if len(splitArray) > 1 {
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

func SecretBuilder(secretManglerObject *v1alpha1.SecretMangler, r *SecretManglerReconciler, ctx context.Context) *v1.Secret {
	log := log.FromContext(ctx)

	newData := map[string][]byte{}

	for newField, newFieldValue := range secretManglerObject.Spec.SecretTemplate.Mappings {
		fmt.Println("newField:", newField, "newFieldValue:", newFieldValue)

		// check if value should be treated as a lookupString
		if strings.HasPrefix(newFieldValue, "<") && strings.HasSuffix(newFieldValue, ">") {
			fmt.Printf("value of field %s indicates a dynamic field\n", newField)

			namespaceName, existingSecretName, existingSecretField, ok := parseLookupString(newFieldValue)
			if ok == false {
				logMsg := fmt.Sprintf("dynamic mapping %s contains a faulty lookup string %s", newField, newFieldValue)
				// FIXME log correctly
				// log.Error(logMsg)
				fmt.Println(logMsg)
				return nil
			}

			// use the namespace of the CR if no explicit namespace is set to lookup existing secret
			if namespaceName == "" {
				namespaceName = secretManglerObject.Namespace
			}

			// fetch secret
			var existingSecret v1.Secret
			namespacedNameExistingSecret := types.NamespacedName{Namespace: namespaceName, Name: existingSecretName}
			if err := r.Get(ctx, namespacedNameExistingSecret, &existingSecret); err != nil {
				// FIXME handle special cases
				logMsg := fmt.Sprintf("unable to fetch secret %s/%s", namespaceName, existingSecretName)
				log.Error(err, logMsg)
				return nil
			}

			// https://stackoverflow.com/a/2050629
			if existingSecretFieldValue, found := existingSecret.Data[existingSecretField]; found {
				fmt.Printf("will add %s: %s to newData ..\n", newField, existingSecretField)
				newData[newField] = existingSecretFieldValue
			}
		} else {
			fmt.Printf("will add %s: %s to newData ..\n", newField, newFieldValue)

			// fixed value can be added as is
			newData[newField] = []byte(newFieldValue)
		}

		fmt.Println("----")
	}

	fmt.Println(newData)

	return &v1.Secret{
		ObjectMeta: v12.ObjectMeta{
			Name:      secretManglerObject.Spec.SecretTemplate.Name,
			Namespace: secretManglerObject.Spec.SecretTemplate.Namespace,
			// Labels: secretManglerObject.Spec.SecretTemplate.Label,*
		},
		Data: newData,
		Type: "Opaque",
	}
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
		Complete(r)
}
