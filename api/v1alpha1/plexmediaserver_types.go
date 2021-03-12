/*
Copyright Adam B Kaplan

SPDX-License-Identifier: Apache-2.0
*/

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
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

	// Storage configures the persistent volume claim attributes for Plex Media Server's backing
	// volumes:
	//
	// 1. Config - Plex's configuration database
	// 2. Transcode - Plex's space for transcoded media files
	// 3. Data - Plex's volume for user-provided media
	// +optional
	Storage PlexStorageSpec `json:"storage,omitempty"`

	// Networking configures network options for the Plex Media Server, such as an external-facing service.
	// +optional
	Networking PlexNetworkSpec `json:"networking,omitempty"`
}

// PlexStorageSpec defines persistent volume claim attributes for the
// volumes used by the Plex Media Server
type PlexStorageSpec struct {

	// Config specifics the volume claim attributes for Plex Media Server's database
	// +optional
	Config *PlexStorageOptions `json:"config,omitempty"`

	// Transcode specifies the volume claim attributes for Plex Media Server's transcoded
	// media files
	// +optional
	Transcode *PlexStorageOptions `json:"transcode,omitempty"`

	// Data specifies the volume claim attributes for Plex Media Server's media data
	// +optional
	Data *PlexStorageOptions `json:"data,omitempty"`
}

// PlexStorageOptions configures a PersistentVolumeClaim used by the Plex Media Server
type PlexStorageOptions struct {

	// AccessMode sets the access mode for the PersistentVolumeClaim used for this Plex volume.
	// +optional
	AccessMode corev1.PersistentVolumeAccessMode `json:"accessMode,omitempty"`

	// Capacity specifies the requested capacity for the PersistentVolumeClaim.
	// The provided volume for this claim may exceed this value.
	// +optional
	Capacity resource.Quantity `json:"capacity,omitempty"`

	// StorageClassName specifies the storage class for the PersistentVolumeClaim.
	// +optional
	StorageClassName *string `json:"storageClassName,omitempty"`

	// Selector is a label selector that can be applied to the PersistentVolumeClaim
	// +optional
	Selector *metav1.LabelSelector `json:"selector,omitempty"`
}

// PlexNetworkSpec specifies network options for the Plex Media Server
type PlexNetworkSpec struct {

	// ExternalServiceType configures an external-facing Service for Plex Media Server, in addition
	// to the headless service used for Plex's underlying StatefulSet deployment.
	// Can be one of NodePort or LoadBalancer
	// +optional
	// +kubebuilder:validation:Enum=NodePort;LoadBalancer
	ExternalServiceType corev1.ServiceType `json:"externalServiceType,omitempty"`

	// EnableDiscovery opens ports necessary for GDM network discovery
	// +optional
	EnableDiscovery bool `json:"enableDiscovery,omitempty"`

	// EnableDLNA opens DLNA access ports on all services.
	// +optional
	EnableDLNA bool `json:"enableDNLA,omitempty"`

	// EnableRoku opens Plex Companion ports used to access Plex via Roku devices.
	// +optional
	EnableRoku bool `json:"enableRoku,omitempty"`
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
