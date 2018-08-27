package v1

import (
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Definition of our CRD AppExternalNat 
type AppExternalNat struct {
	meta_v1.TypeMeta   `json:",inline"`
	meta_v1.ObjectMeta `json:"metadata"`
	Spec               AppExternalNatSpec   `json:"spec"`
	Status             AppExternalNatStatus `json:"status,omitempty"`
}

type AppExternalNatSpec struct {
	IP			string	`json:"ip"`
	Port		string	`json:"port"`
	Protocol	string	`json:"protocol"`
	Rules		[]AppExternalNatRule	`json:"rules"`
}

type AppExternalNatRule struct {
	Host		string	`json:"host"`
	PoolName	string	`json:"pool"`
}

type AppExternalNatStatus struct {
	State   string `json:"state,omitempty"`
	Message string `json:"message,omitempty"`
}

type AppExternalNatList struct {
	meta_v1.TypeMeta `json:",inline"`
	meta_v1.ListMeta `json:"metadata"`
	Items            []AppExternalNat `json:"items"`
}