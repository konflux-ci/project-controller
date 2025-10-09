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

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// Provide a value for a variable specified in an associated
// ProjectDevelopmentStreamTemplate
// Use values to customize generated resources per development stream.
type ProjectDevelopmentStreamSpecTemplateValue struct {
	// The name of the template variable to provide a value for
	Name string `json:"name"`
	// The value to be placed in the template variable
	Value string `json:"value"`
}

// ProjectDevelopmentStreamSpecTemplateRef defines which optional template is
// associated with this ProjectDevelopmentStream and how to apply it
// The template must exist in the same namespace as the stream.
type ProjectDevelopmentStreamSpecTemplateRef struct {
	// The name of the ProjectDevelopmentStreamTemplate to use
	Name string `json:"name"`
	// Values for template variables
	Values []ProjectDevelopmentStreamSpecTemplateValue `json:"values,omitempty"`
}

// ProjectDevelopmentStreamSpec defines the desired state of ProjectDevelopmentStream
// A development stream typically represents a version or environment branch.
type ProjectDevelopmentStreamSpec struct {
	// The name of the project this stream belongs to
	Project string `json:"project,omitempty"`
	// An optional template to use for creating resources owned by this
	// ProjectDevelopmentStream
	Template *ProjectDevelopmentStreamSpecTemplateRef `json:"template,omitempty"`
}

// ProjectDevelopmentStreamStatus defines the observed state of ProjectDevelopmentStream
// Conditions include:
// - Ready (reasons: Reconciling, UpdatingOwnerRef, NoTemplate, TemplateFetchFailed, TemplateGenerationFailed, ResourcesApplied, ApplyingResources)
type ProjectDevelopmentStreamStatus struct {
	// Represents the observations of a ProjectDevelopmentStream's current state.
	// Known .status.conditions.type are: "Ready"
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ProjectDevelopmentStream represents an independent stream of development.
// No custom labels or annotations on the object alter controller behavior.
type ProjectDevelopmentStream struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProjectDevelopmentStreamSpec   `json:"spec,omitempty"`
	Status ProjectDevelopmentStreamStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ProjectDevelopmentStreamList contains a list of ProjectDevelopmentStream
type ProjectDevelopmentStreamList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProjectDevelopmentStream `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ProjectDevelopmentStream{}, &ProjectDevelopmentStreamList{})
}
