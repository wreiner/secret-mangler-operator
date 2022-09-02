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

var _ = Describe("SecretMangler object different namespace KeepLostSync", func() {

	const (
		SecretManglerName      = "base-mangler"
		SecretManglerNamespace = "sns-mns-kls"

		NewSecretName          = "new-secret"
		NewSecretNameNamespace = "sns-mns-kls"

		FirstReferenceSecretName          = "first-ref-secret"
		FirstReferenceSecretNameNamespace = "sns-mns-kls"

		SecondReferenceSecretName          = "second-ref-secret"
		SecondReferenceSecretNameNamespace = "sns-mns-kls-2"

		timeout  = time.Second * 10
		duration = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("When creating a SecretMangler object with multiple reference secrets in multiple namespaces for KeepLostSync", func() {
		It("Should create a new secret with parts of the reference secrets", func() {

			ctx := context.Background()

			By("By creating a new namespace")
			firstNameSpace := &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: FirstReferenceSecretNameNamespace,
				},
			}
			Expect(k8sClient.Create(ctx, firstNameSpace)).Should(Succeed())

			By("By creating a second namespace")
			secondNameSpace := &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: SecondReferenceSecretNameNamespace,
				},
			}
			Expect(k8sClient.Create(ctx, secondNameSpace)).Should(Succeed())

			By("By creating a new reference secret")
			firstReferenceSecret := &v1.Secret{
				ObjectMeta: v12.ObjectMeta{
					Name:      FirstReferenceSecretName,
					Namespace: FirstReferenceSecretNameNamespace,
				},
				Data: map[string][]byte{
					"test": []byte("ZGVydGVzdGRlcg=="),
				},
				Type: "Opaque",
			}
			Expect(k8sClient.Create(ctx, firstReferenceSecret)).Should(Succeed())

			By("By creating a second reference secret")
			secondReferenceSecret := &v1.Secret{
				ObjectMeta: v12.ObjectMeta{
					Name:      SecondReferenceSecretName,
					Namespace: SecondReferenceSecretNameNamespace,
				},
				Data: map[string][]byte{
					"test-2": []byte("ZGVydGVzdGRlcg=="),
				},
				Type: "Opaque",
			}
			Expect(k8sClient.Create(ctx, secondReferenceSecret)).Should(Succeed())

			By("By creating a SecretMangler object")
			lookupString := fmt.Sprintf("<%s/%s:%s>", SecondReferenceSecretNameNamespace, SecondReferenceSecretName, "test-2")
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
						CascadeMode: "KeepLostSync",
						Mappings: map[string]string{
							"dynamicmapping":  "<first-ref-secret:test>",
							"dynamicmapping2": lookupString,
							"fixedmapping":    "fixed-test",
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, secretManglerObject)).Should(Succeed())

			newSecretLookupKey := types.NamespacedName{Name: NewSecretName, Namespace: NewSecretNameNamespace}
			newSecret := &v1.Secret{}

			// build testmap to test created secret
			testmap := make(map[string][]byte)
			testmap["dynamicmapping"] = []byte("ZGVydGVzdGRlcg==")
			testmap["dynamicmapping2"] = []byte("ZGVydGVzdGRlcg==")
			testmap["fixedmapping"] = []byte("fixed-test")

			// We'll need to retry getting this newly created Secret, given that creation may not immediately happen.
			Eventually(func() bool {
				err := k8sClient.Get(ctx, newSecretLookupKey, newSecret)
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())
			Expect(reflect.DeepEqual(testmap, newSecret.Data)).Should(BeTrue())

			// Remove one secret and check that the lost sync data is still present
			Expect(k8sClient.Delete(ctx, secondReferenceSecret)).Should(Succeed())

			newSecret = &v1.Secret{}
			Eventually(func() bool {
				secretManglerLookup := types.NamespacedName{Name: SecretManglerName, Namespace: SecretManglerNamespace}
				err := k8sClient.Get(ctx, secretManglerLookup, secretManglerObject)
				if err != nil {
					return false
				}
				if secretManglerObject.Status.LastAction != "KeepLostSync" {
					return false
				}
				err = k8sClient.Get(ctx, newSecretLookupKey, newSecret)
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())
			Expect(reflect.DeepEqual(testmap, newSecret.Data)).Should(BeTrue())

			// cleanup
			Expect(k8sClient.Delete(ctx, secretManglerObject)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, firstReferenceSecret)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, firstNameSpace)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, secondNameSpace)).Should(Succeed())
		})
	})
})
