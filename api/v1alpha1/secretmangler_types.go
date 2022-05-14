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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SecretManglerSpec defines the desired state of SecretMangler
type SecretManglerSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of SecretMangler. Edit secretmangler_types.go to remove/update
	SecretTemplate SecretTemplateStruct `json:"secretTemplate"`
}

// SecretManglerStatus defines the observed state of SecretMangler
type SecretManglerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

type SecretTemplateStruct struct {
	Name       string `json:"name"`
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Label      string `json:"label,omitempty"`
	Namespace  string `json:"namespace"`
	// Label      metav1.LabelSelector `json:"label,omitempty"`
	// Namespace  metav1.LabelSelector `json:"namespace"`
	Annotation map[string]string `json:"annotation,omitempty"`
	Mappings   map[string]string `json:"mappings"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// SecretMangler is the Schema for the secretmanglers API
type SecretMangler struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SecretManglerSpec   `json:"spec,omitempty"`
	Status SecretManglerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// SecretManglerList contains a list of SecretMangler
type SecretManglerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SecretMangler `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SecretMangler{}, &SecretManglerList{})
}
