/*

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
// +kubebuilder:docs-gen:collapse=Apache License

package controllers

import (
	"context"
	"fmt"
	"reflect"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/wreiner/secret-mangler-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// +kubebuilder:docs-gen:collapse=Imports

var _ = Describe("SecretMangler object different namespace KeepNoAction", func() {

	const (
		SecretManglerName      = "base-mns-mangler"
		SecretManglerNamespace = "default"

		NewSecretName          = "new-mns-secret"
		NewSecretNameNamespace = "default"

		ReferenceSecretName      = "reference-mns-secret"
		ReferenceSecretNamespace = "sm-test-ns"

		timeout  = time.Second * 10
		duration = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("When creating a SecretMangler object with the reference in a different namespace", func() {
		It("Should create a new secret with parts of the reference-secret", func() {

			// build testmap to test created secret
			testmap := make(map[string][]byte)
			testmap["dynamicmapping"] = []byte("ZGVydGVzdGRlcg==")
			testmap["fixedmapping"] = []byte("fixed-test")

			ctx := context.Background()

			By("By creating a new namespace")
			newNameSpace := &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: ReferenceSecretNamespace,
				},
			}
			Expect(k8sClient.Create(ctx, newNameSpace)).Should(Succeed())

			By("By creating a new reference secret")
			referenceSecret := &v1.Secret{
				ObjectMeta: v12.ObjectMeta{
					Name:      ReferenceSecretName,
					Namespace: ReferenceSecretNamespace,
				},
				Data: map[string][]byte{
					"test": []byte("ZGVydGVzdGRlcg=="),
				},
				Type: "Opaque",
			}
			Expect(k8sClient.Create(ctx, referenceSecret)).Should(Succeed())

			By("By creating a SecretMangler object")
			dynamicLookupString := fmt.Sprintf("<%s/%s:test>", ReferenceSecretNamespace, ReferenceSecretName)

			secretManglerObject := &v1alpha1.SecretMangler{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "secret-mangler.wreiner.at/v1alpha1",
					Kind:       "SecretMangler",
				},
				ObjectMeta: v12.ObjectMeta{
					Name:      SecretManglerName,
					Namespace: SecretManglerNamespace,
				},
				Spec: v1alpha1.SecretManglerSpec{
					SecretTemplate: v1alpha1.SecretTemplateStruct{
						APIVersion:  "v1",
						Kind:        "Secret",
						Name:        NewSecretName,
						Namespace:   NewSecretNameNamespace,
						CascadeMode: "KeepNoAction",
						Mappings: map[string]string{
							"dynamicmapping": dynamicLookupString,
							"fixedmapping":   "fixed-test",
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, secretManglerObject)).Should(Succeed())

			newSecretLookupKey := types.NamespacedName{Name: NewSecretName, Namespace: NewSecretNameNamespace}
			newSecret := &v1.Secret{}

			// We'll need to retry getting this newly created Secret, given that creation may not immediately happen.
			Eventually(func() bool {
				err := k8sClient.Get(ctx, newSecretLookupKey, newSecret)
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())
			Expect(reflect.DeepEqual(testmap, newSecret.Data)).Should(BeTrue())

			// cleanup
			Expect(k8sClient.Delete(ctx, secretManglerObject)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, referenceSecret)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, newNameSpace)).Should(Succeed())
		})
	})
})
