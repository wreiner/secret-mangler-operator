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
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/wreiner/secret-mangler-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

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
	// remove unneeded characters
	newFieldValue := strings.TrimLeft(lookupString, "<")
	newFieldValue = strings.TrimRight(newFieldValue, ">")

	// split by / indicates a provided namespace of the secret to lookup
	splitArray := strings.Split(newFieldValue, "/")
	if len(splitArray) > 1 {
		namespaceName = splitArray[0]
		newFieldValue = splitArray[1]
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
		// fmt.Printf("got [%s: %b] ..\n", checkKey, checkValue)

		// https://stackoverflow.com/a/36463704
		if val, ok := (*newData)[checkKey]; ok {
			// fmt.Printf("found key [%s: %b] in newData\n", checkKey, val)

			// key is found but values are different so we need an update
			if comp := bytes.Compare(val, checkValue); comp != 0 {
				needUpdate = true
			}

			continue
		}

		if secretManglerObject.Spec.SecretTemplate.CascadeMode == "KeepLostSync" {
			logMsg = fmt.Sprintf("keeping key %s because of KeepLostSync", checkKey)
			log.Info(logMsg)

			// keep old data which was lost in this reconcile run
			(*newData)[checkKey] = checkValue
			needUpdate = true

			secretManglerObject.Status.LastAction = "KeepLostSync"

		} else if secretManglerObject.Spec.SecretTemplate.CascadeMode == "RemoveLostSync" {
			// just log the message
			logMsg = fmt.Sprintf("removing key %s from data because of RemoveLostSync", checkKey)
			log.Info(logMsg)
			needUpdate = true

			secretManglerObject.Status.LastAction = "RemoveLostSync"

		} else if secretManglerObject.Spec.SecretTemplate.CascadeMode == "CascadeDelete" {
			logMsg = fmt.Sprintf("removing complete secret because of CascadeDelete")
			log.Info(logMsg)

			secretManglerObject.Status.LastAction = "CascadeDelete"

			// secret should be deleted
			return 2
		}
	}

	// sanity check - if newData is empty delete the secret as there is no more data to store in the secret
	if len(*newData) == 0 {
		logMsg = fmt.Sprintf("removing complete secret because there is no data to store")
		log.Info(logMsg)
		return 2
	} else {
		// if newData len is bigger than existingSecretData there is new data
		// which was not in the existing before, most probably because of RemoveLostSync
		if len(*newData) > len(*existingSecretData) {
			needUpdate = true
		}
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
