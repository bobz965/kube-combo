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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// VpnGwSpec defines the desired state of VpnGw
type VpnGwSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// pod request limit cpu, memory
	// 1 cpu at least
	// 1G memory at least
	Cpu    string `json:"cpu"`
	Memory string `json:"memory"`

	// 1Mbps bandwidth at least
	QoSBandwidth string `json:"qosBandwidth"`

	// vpn gw static ip
	Ip string `json:"ip"`
	// vpn gw static public ip, floating ip or router lb ip
	PublicIp string `json:"publicIp"`
	// pod subnet
	// the vpn gw server pod running inside in this pod
	// user can access all pod in this subnet via vpn gw
	// if use subnet lb vip in this subnet, so no need svc cidr in this case
	// vpc subnet use as eth1
	Subnet string `json:"subnet"`
	// pubblic subnet use as eth0
	PublicSubnet string `json:"publicSubnet"`

	Replicas int32 `json:"replicas"`
	// vpn gw pod node selector
	Selector []string `json:"selector,omitempty"`
	// vpn gw pod tolerations
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
	// vpn gw pod affinity
	Affinity corev1.Affinity `json:"affinity,omitempty"`

	// vpn gw enable ssl vpn
	EnableSslVpn bool `json:"enableSslVpn"`
	// ssl vpn use openvpn server
	// all ssl vpn spec start with ovpn

	// ssl vpn secret name, the secret should in the same namespace as the vpn gw
	SslSecret string `json:"sslSecret,omitempty"`

	// ovpn ssl vpn proto, udp or tcp, udp probably is better
	OvpnCipher string `json:"ovpnCipher"`
	OvpnProto  string `json:"ovpnProto"`
	// ovpn ssl vpn port, default 1194 for udp, 443 for tcp
	OvpnPort int `json:"ovpnPort"`
	// ovpn ssl vpn clinet server subnet cidr 10.240.0.0/16
	OvpnSubnetCidr string `json:"ovpnSubnetCidr"`
	// if use kube-ovn default subnet, svc cidr probably is different, should be set
	// pod svc cidr 10.96.0.0/20
	// OvpnSvcCidr string `json:"ovpnSslVpnSvcCidr"`
	// ssl vpn server image, openvpn server
	SslVpnImage string `json:"sslVpnImage"`

	// vpn gw enable ipsec vpn
	EnableIpsecVpn bool `json:"enableIpsecVpn"`
	// ipsec use strongswan server
	// all ipsec vpn spec start with ipsec

	// ipsec vpn secret name, the secret should in the same namespace as the vpn gw
	IpsecSecret string `json:"ipsecSecret,omitempty"`

	// remote ipsec vpn server ips
	IpsecRemoteAddrs string `json:"ipsecRemoteAddrs"`

	// remote ipsec vpn server subnet cidrs
	IpsecRemoteTs string `json:"ipsecRemoteTs"`

	// ipsec vpn server image, strongswan server
	IpsecVpnImage string `json:"ipsecVpnImage"`
}

// VpnGwStatus defines the observed state of VpnGw
type VpnGwStatus struct {
	Cpu              string              `json:"cpu" patchStrategy:"merge"`
	Memory           string              `json:"memory" patchStrategy:"merge"`
	QoSBandwidth     string              `json:"qosBandwidth" patchStrategy:"merge"`
	Ip               string              `json:"ip" patchStrategy:"merge"`
	PublicIp         string              `json:"publicIp" patchStrategy:"merge"`
	Subnet           string              `json:"subnet" patchStrategy:"merge"`
	PublicSubnet     string              `json:"publicSubnet" patchStrategy:"merge"`
	Replicas         int32               `json:"replicas" patchStrategy:"merge"`
	Selector         []string            `json:"selector,omitempty" patchStrategy:"merge"`
	Tolerations      []corev1.Toleration `json:"tolerations,omitempty" patchStrategy:"merge"`
	Affinity         corev1.Affinity     `json:"affinity,omitempty" patchStrategy:"merge"`
	EnableSslVpn     bool                `json:"enableSslVpn" patchStrategy:"merge"`
	SslSecret        string              `json:"sslSecret"  patchStrategy:"merge"`
	SslVpnImage      string              `json:"sslVpnImage" patchStrategy:"merge"`
	OvpnCipher       string              `json:"ovpnCipher" patchStrategy:"merge"`
	OvpnProto        string              `json:"ovpnProto" patchStrategy:"merge"`
	OvpnPort         int                 `json:"ovpnPort" patchStrategy:"merge"`
	OvpnSubnetCidr   string              `json:"ovpnSubnetCidr" patchStrategy:"merge"`
	EnableIpsecVpn   bool                `json:"enableIpsecVpn" patchStrategy:"merge"`
	IpsecSecret      string              `json:"ipsecSecret"  patchStrategy:"merge"`
	IpsecVpnImage    string              `json:"ipsecVpnImage" patchStrategy:"merge"`
	IpsecRemoteAddrs string              `json:"ipsecRemoteAddrs" patchStrategy:"merge"`
	IpsecRemoteTs    string              `json:"ipsecRemoteTs" patchStrategy:"merge"`

	// Conditions store the status conditions of the vpn gw instances
	// +operator-sdk:csv:customresourcedefinitions:type=status
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion
//+kubebuilder:printcolumn:name="Subnet",type=string,JSONPath=`.status.subnet`
//+kubebuilder:printcolumn:name="IP",type=string,JSONPath=`.status.ip`
//+kubebuilder:printcolumn:name="PublicIP",type=string,JSONPath=`.status.publicIp`
//+kubebuilder:printcolumn:name="SSLVPN",type=string,JSONPath=`.status.sslVpnGwEnable`
// +kubebuilder:printcolumn:name="OvpnCipher",type=string,JSONPath=`.status.ovpnCipher`
//+kubebuilder:printcolumn:name="IPSecVPN",type=string,JSONPath=`.status.ipsecVpnGwEnable`

// VpnGw is the Schema for the vpngws API
type VpnGw struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VpnGwSpec   `json:"spec,omitempty"`
	Status VpnGwStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// VpnGwList contains a list of VpnGw
type VpnGwList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VpnGw `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VpnGw{}, &VpnGwList{})
}
