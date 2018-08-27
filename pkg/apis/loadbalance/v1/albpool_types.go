package lbv1

import (
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Definition of our CRD CAppLoadBalancePool class
type CAppLoadBalancePool struct {
	meta_v1.TypeMeta   `json:",inline"`
	meta_v1.ObjectMeta `json:"metadata"`
	Spec               CAppLoadBalancePoolSpec   `json:"spec"`
	Status             CAppLoadBalancePoolStatus `json:"status,omitempty"`
}

type CAppLoadBalancePoolSpec struct {
	Members		[]CAppLoadBalancePoolMember		`json:"members"`
}

type CAppLoadBalancePoolMember struct {
	IP		string	`json:"ip"`
	Port	string	`json:"port"`
	Weight	string	`json:"weight,omitempty"`
}

type CAppLoadBalancePoolStatus struct {
	State   string `json:"state,omitempty"`
	Message string `json:"message,omitempty"`
}

type CAppLoadBalancePoolList struct {
	meta_v1.TypeMeta `json:",inline"`
	meta_v1.ListMeta `json:"metadata"`
	Items            []CAppLoadBalancePool `json:"items"`
}