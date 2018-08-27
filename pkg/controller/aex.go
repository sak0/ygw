package controller

import (
	"time"
	"os"
	"reflect"
	
	"github.com/golang/glog"
	
	"k8s.io/client-go/kubernetes"	
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	meta_v1 	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	
	crdclient 	"github.com/sak0/ygw/pkg/client"
	crdv1 		"github.com/sak0/ygw/pkg/apis/external/v1"
	driver 		"github.com/sak0/ygw/pkg/drivers"
	"github.com/sak0/ygw/pkg/utils"
)

type AexController struct {
	crdClient		*rest.RESTClient
	crdScheme		*runtime.Scheme
	client			kubernetes.Interface
	
	aexController	cache.Controller
	driver			driver.GwProvider
}

func NewAexController(client kubernetes.Interface, crdClient *rest.RESTClient, 
					crdScheme *runtime.Scheme)(*AexController, error) {
	aexctr := &AexController{
		crdClient 	: crdClient,
		crdScheme 	: crdScheme,
		client		: client,
	}
	driver, err := driver.New("f5")
	if err != nil {
		glog.Errorf("Intialization bigip connection faild: %v", err)
		return nil, err
	}
	aexctr.driver = driver	
	
	aexListWatch := cache.NewListWatchFromClient(aexctr.crdClient, 
		crdv1.AEXPlural, meta_v1.NamespaceAll, fields.Everything())
	
	_, aexcontroller := cache.NewInformer(
		aexListWatch,
		&crdv1.AppExternalNat{},
		time.Minute*10,
		cache.ResourceEventHandlerFuncs{
			AddFunc: aexctr.onAexAdd,
			DeleteFunc: aexctr.onAexDel,
			UpdateFunc: aexctr.onAexUpdate,
		},
	)
	aexctr.aexController = aexcontroller
	
	return aexctr, nil
}

func (c *AexController)Run(ctx <-chan struct{}) {
	glog.V(2).Infof("Aex Controller starting...")
	go c.aexController.Run(ctx)
	wait.Poll(time.Second, 5*time.Minute, func() (bool, error) {
		return c.aexController.HasSynced(), nil
	})
	if !c.aexController.HasSynced() {
		glog.Errorf("aex informer initial sync timeout")
		os.Exit(1)
	}
}

func (c *AexController)onAexAdd(obj interface{}) {
	glog.V(3).Infof("Add-Aex: %v", obj)
	aex := obj.(*crdv1.AppExternalNat)
	aexName := utils.GenerateAexName(aex.Namespace, aex.Name)
	err := c.driver.CreateVirtualServer("url", aexName, aex.Spec.IP, aex.Spec.Port, aex.Spec.Protocol)
	if err != nil {
		glog.Errorf("CreateVirtualServer failed: %+v\n", err)
		c.updateError(err.Error(), aex)
		return				
	}
	
	for _, rule := range aex.Spec.Rules {
		host := rule.Host
		pool := rule.PoolName
		poolName := utils.GeneratePoolNameEXP(aex.Namespace, pool)
		err := c.driver.VirtualServerBindURL(aexName, host, poolName)
		if err != nil {
			glog.Errorf("VirtualServerBindURL %s(%s) to %s failed: %+v\n", 
				poolName, host, aexName, err)
			c.updateError(err.Error(), aex)
			return			
		}
	}	
}

func (c *AexController)onAexUpdate(oldObj, newObj interface{}) {
	glog.V(3).Infof("Update-Aex: %v -> %v", oldObj, newObj)

	if !reflect.DeepEqual(oldObj, newObj) {
		newAex := newObj.(*crdv1.AppExternalNat)
		oldAex := oldObj.(*crdv1.AppExternalNat)
		
		rulesNew := utils.GetRulesMap(newAex)
		rulesOld := utils.GetRulesMap(oldAex)
		glog.V(2).Infof("rulesNew: %v", rulesNew)
		glog.V(2).Infof("rulesOld: %v", rulesOld)
		if !reflect.DeepEqual(rulesNew, rulesOld) {
			glog.V(2).Infof("Need update Pool configurations.")
			c.updateAex(newAex, rulesOld, rulesNew)
		}					
	}	
}

func (c *AexController)updateAex(aex *crdv1.AppExternalNat, rulesOld, rulesNew map[crdv1.AppExternalNatRule]int){
	vsName := utils.GenerateAexName(aex.Namespace, aex.Name)
	
	for ruleNew, _ := range rulesNew {
		if _, ok := rulesOld[ruleNew]; !ok {
			glog.V(2).Infof("need add rule %v on %s", ruleNew, vsName)
			poolName := utils.GeneratePoolNameEXP(aex.Namespace, ruleNew.PoolName)
			err := c.driver.VirtualServerBindURL(vsName, ruleNew.Host, poolName)
			if err != nil {
				glog.Errorf("VirtualServerBindURL failed %v", err)
			}
		}
	}
	
	for ruleOld, _ := range rulesOld {
		if _, ok := rulesNew[ruleOld]; !ok {
			glog.V(2).Infof("need remove rule %v from %s", ruleOld, vsName)
			poolName := utils.GeneratePoolNameEXP(aex.Namespace, ruleOld.PoolName)
			err := c.driver.VirtualServerUnbindURL(vsName, ruleOld.Host, poolName)
			if err != nil {
				glog.Errorf("VirtualServerUnbindURL failed %v", err)
			}			
		}
	}
}

func (c *AexController)onAexDel(obj interface{}) {
	glog.V(3).Infof("Del-Aex: %v", obj)

	aex := obj.(*crdv1.AppExternalNat)
	
	aexName := utils.GenerateAexName(aex.Namespace, aex.Name)
	err := c.driver.DeleteVirtualServer(aexName)
	if err != nil {
		glog.Errorf("DeleteVirtualServer failed: %+v\n", err)
	}	
}

func (c *AexController)updateError(msg string, aex *crdv1.AppExternalNat) {
	aex.Status.State = crdv1.AEXSTATUSERROR
	aex.Status.Message = msg
	aexclient := crdclient.AexClient(c.crdClient, c.crdScheme, aex.Namespace)
	_, _ = aexclient.Update(aex, aex.Name)
}