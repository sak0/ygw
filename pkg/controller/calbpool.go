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

type CALBPoolController struct {
	crdClient		*rest.RESTClient
	crdScheme		*runtime.Scheme
	client			kubernetes.Interface
	
	calbPoolController	cache.Controller
	driver				driver.GwProvider
}

func NewCALBPoolController(client kubernetes.Interface, crdClient *rest.RESTClient, 
					crdScheme *runtime.Scheme)(*CALBPoolController, error) {
	calbpctr := &CALBPoolController{
		crdClient 	: crdClient,
		crdScheme 	: crdScheme,
		client		: client,
	}
	driver, _ := driver.New("f5")
	calbpctr.driver = driver	
	
	poolListWatch := cache.NewListWatchFromClient(calbpctr.crdClient, 
		lbv1.CALBPPlural, meta_v1.NamespaceAll, fields.Everything())
	
	_, calbpoolcontroller := cache.NewInformer(
		poolListWatch,
		&lbv1.CAppLoadBalancePool{},
		time.Minute*10,
		cache.ResourceEventHandlerFuncs{
			AddFunc: calbpctr.onPoolAdd,
			DeleteFunc: calbpctr.onPoolDel,
			UpdateFunc: calbpctr.onPoolUpdate,
		},
	)
	calbpctr.calbPoolController = calbpoolcontroller
	
	return calbpctr, nil
}

func (c *CALBPoolController)Run(ctx <-chan struct{}) {
	glog.V(2).Infof("CALB Pool Controller starting...")
	go c.calbPoolController.Run(ctx)
	wait.Poll(time.Second, 5*time.Minute, func() (bool, error) {
		return c.calbPoolController.HasSynced(), nil
	})
	if !c.calbPoolController.HasSynced() {
		glog.Errorf("CALB pool informer initial sync timeout")
		os.Exit(1)
	}
}

func (c *CALBPoolController)onPoolAdd(obj interface{}) {
	glog.V(3).Infof("Add-Pool: %v", obj)
}

func (c *CALBPoolController)onPoolUpdate(oldObj, newObj interface{}) {
	glog.V(3).Infof("Update-Pool: %v -> %v", oldObj, newObj)
}

func (c *CALBPoolController)onPoolDel(obj interface{}) {
	glog.V(3).Infof("Del-Pool: %v", obj)
}

func (c *CALBPoolController)updateError(msg string, pool *lbv1.CAppLoadBalancePool) {
	pool.Status.State = lbv1.CALBPOOLSTATUSERROR
	pool.Status.Message = msg
	poolclient := crdclient.CALBPoolClient(c.crdClient, c.crdScheme, pool.Namespace)
	_, _ = poolclient.Update(pool, pool.Name)
}