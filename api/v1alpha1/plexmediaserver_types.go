/*
Copyright Adam B Kaplan

SPDX-License-Identifier: Apache-2.0
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PlexMediaServerSpec defines the desired state of PlexMediaServer
type PlexMediaServerSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Version is the version of Plex Media server deployed on the cluster
	// +optional
	Version string `json:"version,omitempty"`

	// ClaimToken is the claim token needed to register the Plex Media Server
	// +optional
	ClaimToken string `json:"claimToken,omitempty"`
}

// PlexMediaServerStatus defines the observed state of PlexMediaServer
type PlexMediaServerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// ObservedGeneration is the generation last observed by the controller
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions reports the condition of the Plex Media Server
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// PlexMediaServer is the Schema for the plexmediaservers API
type PlexMediaServer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PlexMediaServerSpec   `json:"spec,omitempty"`
	Status PlexMediaServerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PlexMediaServerList contains a list of PlexMediaServer
type PlexMediaServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PlexMediaServer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PlexMediaServer{}, &PlexMediaServerList{})
}
