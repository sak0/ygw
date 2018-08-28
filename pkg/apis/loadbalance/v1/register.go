package lbv1

import (
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	LBGroup			string = "loadbalance.yonghui.cn"	
	LBVersion		string = "v1"	
	
	CALBPPlural		string = "capploadbalancepool"
	FullCALBPName	string = CALBPPlural + "." + LBGroup

	CALBPlural		string = "capploadbalance"
	FullCALBName	string = CALBPlural + "." + LBGroup	
)

var (
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme   = SchemeBuilder.AddToScheme	
)

// Create a Rest client with the new CRD Schema
var SchemeGroupVersion = schema.GroupVersion{Group: LBGroup, Version: LBVersion}

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&CAppLoadBalancePool{},
		&CAppLoadBalancePoolList{},	
		&CAppLoadBalance{},
		&CAppLoadBalanceList{},
	)
	meta_v1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}