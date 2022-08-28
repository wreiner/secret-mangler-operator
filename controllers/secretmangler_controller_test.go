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

var _ = Describe("SecretMangler object", func() {

	const (
		SecretManglerName      = "base-mangler"
		SecretManglerNamespace = "default"

		NewSecretName          = "new-secret"
		NewSecretNameNamespace = "default"

		timeout  = time.Second * 10
		duration = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("When creating a SecretMangler object", func() {
		It("Should create a new secret with parts of the reference-secret", func() {
			By("By creating a new reference secret")
			ctx := context.Background()
			referenceSecret := &v1.Secret{
				ObjectMeta: v12.ObjectMeta{
					Name:      "reference-secret",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"test": []byte("ZGVydGVzdGRlcg=="),
				},
				Type: "Opaque",
			}
			Expect(k8sClient.Create(ctx, referenceSecret)).Should(Succeed())

			By("By creating a SecretMangler object")
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
							"dynamicmapping": "<reference-secret:test>",
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
			Expect(len(newSecret.Data)).Should(Equal(2))
		})
	})
})
