package utils


import (
	"fmt"
	"net"
	"os"
	"strconv"
	"time"	
	
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	
	crdv1 "github.com/sak0/ygw/pkg/apis/external/v1"
	lbv1 "github.com/sak0/ygw/pkg/apis/loadbalance/v1"
)

func WaitCRDReady(clientset apiextensionsclient.Interface, crdName string) error {
	err := Retry(5*time.Second, 20, func() (bool, error) {
		crd, err := clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Get(crdName, meta_v1.GetOptions{})
		if err != nil {
			return false, err
		}
		for _, cond := range crd.Status.Conditions {
			switch cond.Type {
			case apiextensionsv1beta1.Established:
				if cond.Status == apiextensionsv1beta1.ConditionTrue {
					return true, nil
				}
			case apiextensionsv1beta1.NamesAccepted:
				if cond.Status == apiextensionsv1beta1.ConditionFalse {
					return false, fmt.Errorf("Name conflict: %v", cond.Reason)
				}
			}
		}
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("wait CRD created failed: %v", err)
	}
	return nil
}

func GetEndpointMap(ep *v1.Endpoints)map[string]int{
	var ipmap = make(map[string]int)
	
	if len(ep.Subsets) < 1 {
		return ipmap
	}
	for _, epaddr := range ep.Subsets[0].Addresses {
		ip := epaddr.IP
		for _, epport := range ep.Subsets[0].Ports {
			port := strconv.Itoa(int(epport.Port))
			ipstr := ip + ":" + port
			ipmap[ipstr] = 1
		}
	}
	
	return ipmap
}


func InClusterConfig() (*rest.Config, error) {
	// Work around https://github.com/kubernetes/kubernetes/issues/40973
	// See https://github.com/coreos/etcd-operator/issues/731#issuecomment-283804819
	if len(os.Getenv("KUBERNETES_SERVICE_HOST")) == 0 {
		addrs, err := net.LookupHost("kubernetes.default.svc")
		if err != nil {
			panic(err)
		}
		os.Setenv("KUBERNETES_SERVICE_HOST", addrs[0])
	}
	if len(os.Getenv("KUBERNETES_SERVICE_PORT")) == 0 {
		os.Setenv("KUBERNETES_SERVICE_PORT", "443")
	}
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func MustNewKubeClient() kubernetes.Interface {
	cfg, err := InClusterConfig()
	if err != nil {
		panic(err)
	}
	return kubernetes.NewForConfigOrDie(cfg)
}

func GetMembersMap(pool *crdv1.ExternalNatPool)map[string]int {
	memberMap := make(map[string]int)
	if len(pool.Spec.Members) < 1 {
		return memberMap
	}
	
	for _, member := range pool.Spec.Members {
		memberStr := member.IP + ":" + member.Port
		memberMap[memberStr] = 1
	}
	
	return memberMap
}

func GetCALBMembersMap(pool *lbv1.CAppLoadBalancePool)map[string]int {
	memberMap := make(map[string]int)
	if len(pool.Spec.Members) < 1 {
		return memberMap
	}
	
	var weight string
	for _, member := range pool.Spec.Members {
		if member.Weight != "" {
			weight = member.Weight
		} else {
			weight = "1"
		}		
		
		memberStr := member.IP + ":" + member.Port + ":" + weight
		memberMap[memberStr] = 1	
	}
	
	return memberMap
}

func GetRulesMap(aex *crdv1.AppExternalNat)map[crdv1.AppExternalNatRule]int {
	rulesMap := make(map[crdv1.AppExternalNatRule]int)
	if len(aex.Spec.Rules) < 1 {
		return rulesMap
	}
	
	for _, rule := range aex.Spec.Rules{
		rulesMap[rule] = 1
	}
	
	return rulesMap
}

func GetCALBPathsMap(calb *lbv1.CAppLoadBalance)map[string]int{
	pathsMap := make(map[string]int)
	if len(calb.Spec.Rules) < 1 {
		return pathsMap
	} 
	
	for _, rule := range calb.Spec.Rules {
		domainName := rule.Host 
		for _, path := range rule.Paths {
			pathName := path.Path
			if pathName == "" {
				pathName = "nilpath"
			}		
			poolName := path.Pool
			keyStr := domainName + "_" + pathName + "_" + poolName
			pathsMap[keyStr] = 1
		}
	}
	return pathsMap
}