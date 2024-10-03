/*
Copyright 2024.

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

package controller

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/wreiner/secret-mangler-operator/api/v1alpha1"
	secretmanglerwreineratv1alpha1 "github.com/wreiner/secret-mangler-operator/api/v1alpha1"
)

// SecretManglerReconciler reconciles a SecretMangler object
type SecretManglerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=secret-mangler.wreiner.at,resources=secretmanglers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=secret-mangler.wreiner.at,resources=secretmanglers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=secret-mangler.wreiner.at,resources=secretmanglers/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=secrets/status,verbs=get

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
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

	msg := fmt.Sprintf("received reconcile request ..")
	log.Info(msg)

	existingSecret := RetrieveSecret(secretMangler.Spec.SecretTemplate.Name, secretMangler.Spec.SecretTemplate.Namespace, r, ctx)
	if existingSecret == nil {
		// create secret on the cluster
		log.Info("did not find existing secret, will try to create new secret ..")

		// build the secret
		newSecret := SecretBuilder(&secretMangler, nil, r, ctx)
		if newSecret == nil {
			msg = fmt.Sprintf("building the secret failed ..")
			log.Info(msg)
			return ctrl.Result{}, nil
		}
		log.Info("after builder")

		msg = fmt.Sprintf("will create secret ..")
		log.Info(msg)

		if err := r.Create(ctx, newSecret); err != nil {
			log.Error(err, "unable to create secret for SecretMangler")
			return ctrl.Result{}, err
		}

		secretMangler.Status.SecretCreated = true
		secretMangler.Status.LastAction = "Create"
	} else {
		// work on a previously created secret
		log.Info("found existing secret, will check fields ..")

		cascadeMode := secretMangler.Spec.SecretTemplate.CascadeMode

		// with KeepNoAction the existing secret which was created on an earlier run will be kept as is
		// KeepNoAction is also the default behaviour if cascadeMode is not set.
		if cascadeMode == "" || cascadeMode == "KeepNoAction" {
			msg = fmt.Sprintf("will not attempt sync because cascadeMode KeepNoAction ..")
			log.Info(msg)

			return ctrl.Result{}, nil
		}

		// get updated secret data
		newData := make(map[string][]byte)
		ok := DataBuilder(&secretMangler, &newData, false, r, ctx)
		if ok == false {
			msg = fmt.Sprintf("building secret data failed.")
			log.Info(msg)
			return ctrl.Result{}, nil
		}

		actionIndicator := CompareExistingSecretDataToNewData(&secretMangler, &existingSecret.Data, &newData, ctx)
		switch actionIndicator {
		case 0:
			// nothing todo
			msg = fmt.Sprintf("secret data has not changed")
			log.Info(msg)
			return ctrl.Result{}, nil

		case 1:
			// update needed
			msg = fmt.Sprintf("secret data has changed, will update ..")
			log.Info(msg)

			// build the secret
			newSecret := SecretBuilder(&secretMangler, &newData, r, ctx)
			if newSecret == nil {
				msg = fmt.Sprintf("building the secret failed")
				log.Info(msg)
				return ctrl.Result{}, nil
			}

			if err := r.Update(ctx, newSecret); err != nil {
				log.Error(err, "unable to update secret")
				return ctrl.Result{}, err
			}

		case 2:
			// delete needed
			msg = fmt.Sprintf("secret will be deleted ..")
			log.Info(msg)

			if err := r.Delete(ctx, existingSecret); err != nil {
				log.Error(err, "unable to delete secret")
				return ctrl.Result{}, err
			}

			secretMangler.Status.SecretCreated = false
		}
	}

	msg = fmt.Sprintf("secret updated, will now update status ..")
	log.Info(msg)

	// update the status
	if err := r.Status().Update(ctx, &secretMangler); err != nil {
		log.Error(err, "unable to update SecretMangler status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SecretManglerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&secretmanglerwreineratv1alpha1.SecretMangler{}).
		Owns(&v1.Secret{}).
		Watches(
			&v1.Secret{},
			handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []ctrl.Request {
				// fmt.Println("in watches function")
				secret, ok := obj.(*v1.Secret)
				if !ok {
					// FIXME
					// fmt.Println("secret is nil?")
					return nil
				}

				// fmt.Printf("Secret [%s/%s] changed, checking for corresponding SecretMangler object ..\n", secret.Namespace, secret.Name)

				var reconcileRequests []reconcile.Request
				secretManglerList := &secretmanglerwreineratv1alpha1.SecretManglerList{}
				client := mgr.GetClient()

				err := client.List(context.TODO(), secretManglerList)
				if err != nil {
					return []reconcile.Request{}
				}

				// fmt.Println("will iterate over SecretMangler objects and their mappings/mirrors ..")
				for _, secretManglerObj := range secretManglerList.Items {
					// fmt.Printf("got SecretMangler object [%s/%s]\n", secretManglerObj.Namespace, secretManglerObj.Name)

					// FIXME needed when mirror is implemented
					// if secretManglerObj.Spec.SecretTemplate.Mirror != "" {

					// }
					if len(secretManglerObj.Spec.SecretTemplate.Mappings) != 0 {
						for field, fieldValue := range secretManglerObj.Spec.SecretTemplate.Mappings {
							if IsLookupString(fieldValue) {
								// get namespace and name from the dynamic field
								// if current secret is part of the dynamic field add to reconciliation request
								// fmt.Printf("value [%s] of field %s indicates a dynamic field\n", fieldValue, field)

								referencedSecretNamespaceName, referencedSecretName, _, ok := ParseLookupString(fieldValue)
								if ok == false {
									logMsg := fmt.Sprintf("dynamic mapping %s contains a faulty lookup string %s", field, fieldValue)
									// no ctx for logging available?
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

									// fmt.Printf("found reference to [%s/%s]\n", referencedSecretNamespaceName, referencedSecretName)

									// check if the secret is in the same namespace
									if referencedSecretNamespaceName == secret.Namespace {
										// fmt.Printf("will add SecretMangler object [%s/%s] to reconciliation requests ..\n", secretManglerObj.Namespace, secretManglerObj.Name)
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

				// fmt.Println("will leave watches function")
				// fmt.Println("------ watch ------")
				return reconcileRequests
			}),
		).
		Complete(r)
}
