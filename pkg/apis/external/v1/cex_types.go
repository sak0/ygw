package v1

import (
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Definition of our CRD ClassicExternalNat 
type ClassicExternalNat struct {
	meta_v1.TypeMeta   `json:",inline"`
	meta_v1.ObjectMeta `json:"metadata"`
	Spec               ClassicExternalNatSpec   `json:"spec"`
	Status             ClassicExternalNatStatus `json:"status,omitempty"`
}

type ClassicExternalNatSpec struct {
	IP			string	`json:"ip"`
	Port		string	`json:"port"`
	Protocol		string	`json:"protocol"`
	Backends	[]ClassicExternalNatBackend	`json:"backends"`
}

type ClassicExternalNatBackend struct {
	PoolName	string	`json:"poolName"`
}

type ClassicExternalNatStatus struct {
	State   string `json:"state,omitempty"`
	Message string `json:"message,omitempty"`
}

type ClassicExternalNatList struct {
	meta_v1.TypeMeta `json:",inline"`
	meta_v1.ListMeta `json:"metadata"`
	Items            []ClassicExternalNat `json:"items"`
}