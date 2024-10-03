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

	"github.com/wreiner/secret-mangler-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// DataBuilder generates the data mappings of a secret from a SecretMangler object.
func DataBuilder(secretManglerObject *v1alpha1.SecretMangler, newData *map[string][]byte, returnOnSourceNotFound bool, r *SecretManglerReconciler, ctx context.Context) bool {
	log := log.FromContext(ctx)

	if newData == nil {
		logMsg := "provided newdata map is nil in DataBuilder, data cannot be build .."
		log.Info(logMsg)
		return false
	}

	for newField, newFieldValue := range secretManglerObject.Spec.SecretTemplate.Mappings {
		// fmt.Println("newField:", newField, "newFieldValue:", newFieldValue)

		// check if value should be treated as a lookupString
		if IsLookupString(newFieldValue) {
			// fmt.Printf("value of field %s indicates a dynamic field\n", newField)

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
				// fmt.Printf("will add %s: %s to newData ..\n", newField, existingSecretFieldValue)
				(*newData)[newField] = existingSecretFieldValue
			}
		} else {
			// fmt.Printf("will add %s: %s to newData ..\n", newField, newFieldValue)

			// fixed value can be added as is
			(*newData)[newField] = []byte(newFieldValue)
		}

		// fmt.Println("----")
	}

	return true
}
