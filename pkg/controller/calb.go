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
	lbv1 		"github.com/sak0/ygw/pkg/apis/loadbalance/v1"
	driver 		"github.com/sak0/ygw/pkg/drivers"
//	"github.com/sak0/ygw/pkg/utils"
)

type CALBController struct {
	crdClient		*rest.RESTClient
	crdScheme		*runtime.Scheme
	client			kubernetes.Interface
	
	calbController	cache.Controller
	driver				driver.GwProvider
}

func NewCALBController(client kubernetes.Interface, crdClient *rest.RESTClient, 
					crdScheme *runtime.Scheme)(*CALBController, error) {
	calbctr := &CALBController{
		crdClient 	: crdClient,
		crdScheme 	: crdScheme,
		client		: client,
	}
	driver, _ := driver.New("f5")
	calbctr.driver = driver	
	
	calbListWatch := cache.NewListWatchFromClient(calbctr.crdClient, 
		lbv1.CALBPlural, meta_v1.NamespaceAll, fields.Everything())
	
	_, calbController := cache.NewInformer(
		calbListWatch,
		&lbv1.CAppLoadBalance{},
		time.Minute*10,
		cache.ResourceEventHandlerFuncs{
			AddFunc: calbctr.onCAlbAdd,
			DeleteFunc: calbctr.onCAlbDel,
			UpdateFunc: calbctr.onCAlbUpdate,
		},
	)
	calbctr.calbController = calbController
	
	return calbctr, nil
}

func (c *CALBController)Run(ctx <-chan struct{}) {
	glog.V(2).Infof("CALB Controller starting...")
	go c.calbController.Run(ctx)
	wait.Poll(time.Second, 5*time.Minute, func() (bool, error) {
		return c.calbController.HasSynced(), nil
	})
	if !c.calbController.HasSynced() {
		glog.Errorf("CALB informer initial sync timeout")
		os.Exit(1)
	}
}

func (c *CALBController)onCAlbAdd(obj interface{}) {
	glog.V(3).Infof("Add-CALB: %v", obj)
}

func (c *CALBController)onCAlbUpdate(oldObj, newObj interface{}) {
	glog.V(3).Infof("Update-CALB: %v -> %v", oldObj, newObj)
}

func (c *CALBController)onCAlbDel(obj interface{}) {
	glog.V(3).Infof("Del-CALB: %v", obj)
}

func (c *CALBController)updateError(msg string, calb *lbv1.CAppLoadBalance) {
	calb.Status.State = lbv1.CALBSTATUSERROR
	calb.Status.Message = msg
	calbclient := crdclient.CALBClient(c.crdClient, c.crdScheme, calb.Namespace)
	_, _ = calbclient.Update(calb, calb.Name)
}