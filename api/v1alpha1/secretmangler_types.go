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
	SecretCreated bool `json:"secretCreated"`
}

// CascadeMode describes edge cases in handling secret syncing.
// Only one of the following cascacde modes may be specified.
// If none of the following modes is specified, the default one
// is KeepNoAction.
// +kubebuilder:validation:Enum=KeepNoAction;KeepLostSync;RemoveLostSync;CascadeDelete
type CascadeMode string

const (
	// KeepNoAction keeps the secret it was initially created and no sync of
	// changes in referenced secrets is performed.
	KeepNoAction CascadeMode = "KeepNoAction"

	// KeepLostSync tries to sync data from referenced secrets.
	// If one or more sources are lost their data is kept as it was synced last.
	KeepLostSync CascadeMode = "KeepLostSync"

	// RemoveLostSync tries to sync data from referenced secrets.
	// If one or more sources re lost their data will be removed from the
	// created secret.
	// If no more sources are available and no fixed mappings are present the
	// secret will be removed as a whole.
	RemoveLostSync CascadeMode = "RemoveLostSync"

	// CascadeDelete removes the secret entirely if only one source is lost no
	// matter whether other sources are still present or not.
	CascadeDelete CascadeMode = "CascadeDelete"
)

type SecretTemplateStruct struct {
	Name       string `json:"name"`
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Label      string `json:"label,omitempty"`
	Namespace  string `json:"namespace"`
	// Label      metav1.LabelSelector `json:"label,omitempty"`
	// Namespace  metav1.LabelSelector `json:"namespace"`
	Annotation  map[string]string `json:"annotation,omitempty"`
	Mappings    map[string]string `json:"mappings"`
	CascadeMode CascadeMode       `json:"cascadeMode,omitempty"`
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
