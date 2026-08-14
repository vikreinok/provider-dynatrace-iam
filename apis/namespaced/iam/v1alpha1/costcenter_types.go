// SPDX-FileCopyrightText: 2024 The Crossplane Authors <https://crossplane.io>
//
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	v2 "github.com/crossplane/crossplane/apis/v2/core/v2"
)

type CostCenterInitParameters struct {
	// (String) The key of the cost center allocation value.
	Key *string `json:"key,omitempty"`

	// (String) The value of the cost center allocation value.
	Value *string `json:"value,omitempty"`
}

type CostCenterObservation struct {
	// (String) The ID of this resource (same as key).
	ID *string `json:"id,omitempty"`

	// (String) The key of the cost center allocation value.
	Key *string `json:"key,omitempty"`

	// (String) The value of the cost center allocation value.
	Value *string `json:"value,omitempty"`
}

type CostCenterParameters struct {
	// (String) The key of the cost center allocation value.
	// +kubebuilder:validation:Optional
	Key *string `json:"key,omitempty"`

	// (String) The value of the cost center allocation value.
	// +kubebuilder:validation:Optional
	Value *string `json:"value,omitempty"`
}

// CostCenterSpec defines the desired state of CostCenter
type CostCenterSpec struct {
	v2.ManagedResourceSpec `json:",inline"`
	ForProvider            CostCenterParameters `json:"forProvider"`
	// InitProvider holds the same fields as ForProvider, with the exception
	// of Identifier and other resource reference fields.
	InitProvider CostCenterInitParameters `json:"initProvider,omitempty"`
}

// CostCenterStatus defines the observed state of CostCenter.
type CostCenterStatus struct {
	v2.ManagedResourceStatus `json:",inline"`
	AtProvider               CostCenterObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion

// CostCenter is the Schema for the CostCenters API.
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,dynatrace}
type CostCenter struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// +kubebuilder:validation:XValidation:rule="!('*' in self.managementPolicies || 'Create' in self.managementPolicies || 'Update' in self.managementPolicies) || has(self.forProvider.key) || (has(self.initProvider) && has(self.initProvider.key))",message="spec.forProvider.key is a required parameter"
	// +kubebuilder:validation:XValidation:rule="!('*' in self.managementPolicies || 'Create' in self.managementPolicies || 'Update' in self.managementPolicies) || has(self.forProvider.value) || (has(self.initProvider) && has(self.initProvider.value))",message="spec.forProvider.value is a required parameter"
	Spec   CostCenterSpec   `json:"spec"`
	Status CostCenterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CostCenterList contains a list of CostCenters
type CostCenterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CostCenter `json:"items"`
}

// Repository type metadata.
var (
	CostCenter_Kind             = "CostCenter"
	CostCenter_GroupKind        = schema.GroupKind{Group: CRDGroup, Kind: CostCenter_Kind}.String()
	CostCenter_KindAPIVersion   = CostCenter_Kind + "." + CRDGroupVersion.String()
	CostCenter_GroupVersionKind = CRDGroupVersion.WithKind(CostCenter_Kind)
)

func init() {
	SchemeBuilder.Register(&CostCenter{}, &CostCenterList{})
}
