package v1

import (
//	"reflect"
//	"github.com/golang/glog"
//
//	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
//	apiextcs "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
//	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	EXGroup			string = "external.yonghui.cn"	
	EXVersion		string = "v1"	
	
	AEXPlural		string = "appexternalnat"
	FullAEXName		string = AEXPlural + "." + EXGroup

	CEXPlural		string = "classicexternalnat"
	FullCEXName		string = CEXPlural + "." + EXGroup

	EXPPlural		string = "externalnatpool"
	FullEXPName		string = EXPPlural + "." + EXGroup	
)

var (
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme   = SchemeBuilder.AddToScheme	
)

// Create a Rest client with the new CRD Schema
var SchemeGroupVersion = schema.GroupVersion{Group: EXGroup, Version: EXVersion}

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&ExternalNatPool{},
		&ExternalNatPoolList{},
		&ClassicExternalNat{},
		&ClassicExternalNatList{},
	)
	meta_v1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}