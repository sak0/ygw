package lbv1

import (
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Definition of our CRD CAppLoadBalance class
type CAppLoadBalance struct {
	meta_v1.TypeMeta   `json:",inline"`
	meta_v1.ObjectMeta `json:"metadata"`
	Spec               CAppLoadBalanceSpec   `json:"spec"`
	Status             CAppLoadBalanceStatus `json:"status,omitempty"`
}

type CAppLoadBalanceSpec struct {
	IP		string					`json:"ip,omitempty"`
	Port	string					`json:"port,omitempty"`
	Subnet	string					`json:"subnet"`
	Rules	[]CAppLoadBalanceRule	`json:"rules,omitempty"`
}

type CAppLoadBalanceRule struct {
	Host	string					`json:"host,omitempty"`
	Paths	[]CAppLoadBalancePath	`json:"paths,omitempty"`
}

type CAppLoadBalancePath struct {
	Path	string	`json:"path,omitempty"`
	Pool	string	`json:"pool"`
}

type CAppLoadBalanceStatus struct {
	State   string `json:"state,omitempty"`
	Message string `json:"message,omitempty"`
}

type CAppLoadBalanceList struct {
	meta_v1.TypeMeta `json:",inline"`
	meta_v1.ListMeta `json:"metadata"`
	Items            []CAppLoadBalance `json:"items"`
}