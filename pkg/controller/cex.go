package controller

import (
	"time"
	"os"
	
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

type CexController struct {
	crdClient		*rest.RESTClient
	crdScheme		*runtime.Scheme
	client			kubernetes.Interface
	
	cexController	cache.Controller
	driver			driver.GwProvider
}

func NewCexController(client kubernetes.Interface, crdClient *rest.RESTClient, 
					crdScheme *runtime.Scheme)(*CexController, error) {
	cexctr := &CexController{
		crdClient 	: crdClient,
		crdScheme 	: crdScheme,
		client		: client,
	}
	driver, _ := driver.New("f5")
	cexctr.driver = driver	
	
	cexListWatch := cache.NewListWatchFromClient(cexctr.crdClient, 
		crdv1.CEXPlural, meta_v1.NamespaceAll, fields.Everything())
	
	_, cexcontroller := cache.NewInformer(
		cexListWatch,
		&crdv1.ClassicExternalNat{},
		time.Minute*10,
		cache.ResourceEventHandlerFuncs{
			AddFunc: cexctr.onCexAdd,
			DeleteFunc: cexctr.onCexDel,
			UpdateFunc: cexctr.onCexUpdate,
		},
	)
	cexctr.cexController = cexcontroller
	
	return cexctr, nil
}

func (c *CexController)Run(ctx <-chan struct{}) {
	glog.V(2).Infof("Cex Controller starting...")
	go c.cexController.Run(ctx)
	wait.Poll(time.Second, 5*time.Minute, func() (bool, error) {
		return c.cexController.HasSynced(), nil
	})
	if !c.cexController.HasSynced() {
		glog.Errorf("cex informer initial sync timeout")
		os.Exit(1)
	}
}

func (c *CexController)onCexAdd(obj interface{}) {
	glog.V(3).Infof("Add-Cex: %v", obj)
	cex := obj.(*crdv1.ClassicExternalNat)
	
	cexName := utils.GenerateCexName(cex.Namespace, cex.Name)
	err := c.driver.CreateVirtualServer("nat", cexName, cex.Spec.IP, cex.Spec.Port, cex.Spec.Protocol)
	if err != nil {
		glog.Errorf("CreateVirtualServer failed: %+v\n", err)
		c.updateError(err.Error(), cex)
		return				
	}
	
	if len(cex.Spec.Backends) > 0 {
		for _, backend := range cex.Spec.Backends {
			poolName := utils.GeneratePoolNameEXP(cex.Namespace, backend.PoolName)
			err := c.driver.VirtualServerBindPool(cexName, poolName)
			if err != nil {
				glog.Errorf("VirtualServerBindPool failed: %+v\n", err)
				c.updateError(err.Error(), cex)
				return				
			}
		}
	}
}

func (c *CexController)onCexUpdate(oldObj, newObj interface{}) {
	glog.V(3).Infof("Update-Cex: %v -> %v", oldObj, newObj)	
}


func (c *CexController)onCexDel(obj interface{}) {
	glog.V(3).Infof("Del-Cex: %v", obj)
	cex := obj.(*crdv1.ClassicExternalNat)
	
	cexName := utils.GenerateCexName(cex.Namespace, cex.Name)
	err := c.driver.DeleteVirtualServer(cexName)
	if err != nil {
		glog.Errorf("DeleteVirtualServer failed: %+v\n", err)
	}
}

func (c *CexController)updateError(msg string, cex *crdv1.ClassicExternalNat) {
	cex.Status.State = crdv1.CEXSTATUSERROR
	cex.Status.Message = msg
	cexclient := crdclient.CexClient(c.crdClient, c.crdScheme, cex.Namespace)
	_, _ = cexclient.Update(cex, cex.Name)
}