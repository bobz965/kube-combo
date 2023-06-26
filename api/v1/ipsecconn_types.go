/*
Copyright 2023.

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// IpsecConnSpec defines the desired state of IpsecConn
type IpsecConnSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	VpnGw string `json:"vpnGw"`
	// CN is defined in x509 certificate
	LocalCN string `json:"localCN"`
	// current public ipsec vpn gw ip
	LocalPublicIp     string `json:"localPublicIp"`
	LocalPrivateCidrs string `json:"localPrivateCidrs"`

	RemoteCN string `json:"remoteCN"`
	// remote public ipsec vpn gw ip
	RemotePublicIp     string `json:"remotePublicIp"`
	RemotePrivateCidrs string `json:"remotePrivateCidrs"`
}

// IpsecConnStatus defines the observed state of IpsecConn
type IpsecConnStatus struct {
	// ipsec connection belong to which vpn gw, will trigger vpn gw reconcile its ipsec connections
	VpnGw string `json:"vpnGw"`

	// CN is defined in x509 certificate
	LocalCN string `json:"localCN"`
	// current public ipsec vpn gw ip
	LocalPublicIp     string `json:"localPublicIp"`
	LocalPrivateCidrs string `json:"localPrivateCidrs"`

	RemoteCN string `json:"remoteCN"`
	// remote public ipsec vpn gw ip
	RemotePublicIp     string `json:"remotePublicIp"`
	RemotePrivateCidrs string `json:"remotePrivateCidrs"`

	// Conditions store the status conditions of the ipsec connection instances
	// +operator-sdk:csv:customresourcedefinitions:type=status
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="VpnGw",type=string,JSONPath=`.status.vpnGw`
// +kubebuilder:printcolumn:name="LocalPublicIp",type=string,JSONPath=`.status.localPublicIp`
// +kubebuilder:printcolumn:name="RemotePublicIp",type=string,JSONPath=`.status.remotePublicIp`
// +kubebuilder:printcolumn:name="LocalPrivateCidrs",type=string,JSONPath=`.status.localPrivateCidrs`
// +kubebuilder:printcolumn:name="RemotePrivateCidrs",type=string,JSONPath=`.status.remotePrivateCidrs`
// +kubebuilder:printcolumn:name="LocalCN",type=string,JSONPath=`.status.localCN`
// +kubebuilder:printcolumn:name="RemoteCN",type=string,JSONPath=`.status.remoteCN`
// IpsecConn is the Schema for the ipsecconns API
type IpsecConn struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IpsecConnSpec   `json:"spec,omitempty"`
	Status IpsecConnStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// IpsecConnList contains a list of IpsecConn
type IpsecConnList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IpsecConn `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IpsecConn{}, &IpsecConnList{})
}
