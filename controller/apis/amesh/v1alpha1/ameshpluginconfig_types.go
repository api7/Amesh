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

// AmeshPluginConfigType is the config type
type AmeshPluginConfigType string

const (
	AmeshPluginConfigTypePreRequest  = "pre-req"
	AmeshPluginConfigTypePostRequest = "post-req"
)

// AmeshPluginConfigPlugin is the plugin config detail
type AmeshPluginConfigPlugin struct {
	//+kubebuilder:validation:Enum=pre-req;post-req
	Type AmeshPluginConfigType `json:"type,omitempty" yaml:"type,omitempty"`
	//+kubebuilder:validation:Required
	//+kubebuilder:validation:MinLength=1
	Name string `json:"name,omitempty" yaml:"name,omitempty"`
	//+kubebuilder:validation:Required
	//+kubebuilder:validation:MinLength=2
	Config string `json:"config,omitempty" yaml:"config,omitempty"`
}

// AmeshPluginConfigSpec defines the desired state of AmeshPluginConfig
type AmeshPluginConfigSpec struct {
	//+required
	//+kubebuilder:validation:Required
	//+kubebuilder:validation:MinItems=1
	Plugins []AmeshPluginConfigPlugin `json:"plugins,omitempty" yaml:"plugins,omitempty"`

	// Label selector for pods.
	Selector *metav1.LabelSelector `json:"selector,omitempty" yaml:"selector,omitempty"`
}

// AmeshPluginConfigStatus defines the observed state of AmeshPluginConfig
type AmeshPluginConfigStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty" yaml:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+genclient
//+k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
//+kubebuilder:resource:shortName=apc

// AmeshPluginConfig is the Schema for the ameshpluginconfigs API
type AmeshPluginConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AmeshPluginConfigSpec   `json:"spec,omitempty"`
	Status AmeshPluginConfigStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true
//+k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AmeshPluginConfigList contains a list of AmeshPluginConfig
type AmeshPluginConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AmeshPluginConfig `json:"items"`
}
