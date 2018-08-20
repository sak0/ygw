package v1

import (
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Definition of our CRD AppLoadBalance class
type ExternalNatPool struct {
	meta_v1.TypeMeta   `json:",inline"`
	meta_v1.ObjectMeta `json:"metadata"`
	Spec               ExternalNatPoolSpec   `json:"spec"`
	Status             ExternalNatPoolStatus `json:"status,omitempty"`
}

type ExternalNatPoolSpec struct {
	Method		string						`json:"lb_method,omitempty"`
	Members		[]ExternalNatPoolMember		`json:"members"`
}

type ExternalNatPoolMember struct {
	IP		string	`json:"ip,omitempty"`
	Port	string	`json:"port"`
}

type ExternalNatPoolStatus struct {
	State   string `json:"state,omitempty"`
	Message string `json:"message,omitempty"`
}

type ExternalNatPoolList struct {
	meta_v1.TypeMeta `json:",inline"`
	meta_v1.ListMeta `json:"metadata"`
	Items            []ExternalNatPool `json:"items"`
}