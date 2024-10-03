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

	"github.com/wreiner/secret-mangler-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// SecretBuilder generates a secret based on a SecretMangler object with all data and metadata.
// The secret will not be applied to the Kubernetes cluster.
func SecretBuilder(secretManglerObject *v1alpha1.SecretMangler, givenData *map[string][]byte, r *SecretManglerReconciler, ctx context.Context) *v1.Secret {
	log := log.FromContext(ctx)

	// Build the data mappings of the secret if it is not given
	newData := make(map[string][]byte)
	if givenData == nil || len((*givenData)) == 0 {
		log.Info("no data or empty data given to SecretBuilder, trying to obtain data ..")
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
