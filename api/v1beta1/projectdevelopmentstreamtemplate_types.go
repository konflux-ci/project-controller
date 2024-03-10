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

// Settings for a variable to be used to customize the template results
type ProjectDevelopmentStreamTemplateVariable struct {
	// Variable name
	Name string `json:"name"`
	// Optional default value for use when a value for the variable is not given
	// can reference values of other previously defined variables using the Go
	// text/template syntax
	Default *string `json:"default,omitempty"`
	// Optional description for the variable for display in the UI
	Description string `json:"description,omitempty"`
}

// ProjectDevelopmentStreamTemplateSpec defines the resources to be generated
// using a ProjectDevelopmentStreamTemplate
type ProjectDevelopmentStreamTemplateSpec struct {
	// The name of the project this stream template belongs to
	Project string `json:"project,omitempty"`
	// List of variables to allow customizing the template results. The order
	// variables in the list is significant as earlier variables can be
	// referenced by the default values for later variables
	Variables []ProjectDevelopmentStreamTemplateVariable `json:"variables,omitempty"`
	// List of resources to be created for version made from this template
	// certain values for resource properties may include references to
	// variables using the Go-text/template syntax
	Resources []UnstructuredObj `json:"resources,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ProjectDevelopmentStreamTemplate is the Schema for the projectdevelopmentstreamtemplates API
type ProjectDevelopmentStreamTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ProjectDevelopmentStreamTemplateSpec `json:"spec,omitempty"`
}

//+kubebuilder:object:root=true

// ProjectDevelopmentStreamTemplateList contains a list of ProjectDevelopmentStreamTemplate
type ProjectDevelopmentStreamTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProjectDevelopmentStreamTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ProjectDevelopmentStreamTemplate{}, &ProjectDevelopmentStreamTemplateList{})
}
