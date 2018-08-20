package controller

import (
	"time"
	"os"
	
	"github.com/golang/glog"
	
	"k8s.io/client-go/kubernetes"	
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	
	crdv1 "github.com/sak0/ygw/pkg/apis/external/v1"
)

type PoolController struct {
	crdClient		*rest.RESTClient
	crdScheme		*runtime.Scheme
	client			kubernetes.Interface
	
	poolController	cache.Controller
}

func NewPoolController(client kubernetes.Interface, crdClient *rest.RESTClient, 
					crdScheme *runtime.Scheme)(*PoolController, error) {
	poolctr := &PoolController{
		crdClient 	: crdClient,
		crdScheme 	: crdScheme,
		client		: client,
	}
	
	poolListWatch := cache.NewListWatchFromClient(poolctr.crdClient, 
		crdv1.EXPPlural, meta_v1.NamespaceAll, fields.Everything())
	
	_, poolcontroller := cache.NewInformer(
		poolListWatch,
		&crdv1.ExternalNatPool{},
		time.Minute*10,
		cache.ResourceEventHandlerFuncs{
			AddFunc: poolctr.onPoolAdd,
			DeleteFunc: poolctr.onPoolDel,
			UpdateFunc: poolctr.onPoolUpdate,
		},
	)
	poolctr.poolController = poolcontroller
	
	return poolctr, nil
}

func (c *PoolController)Run(ctx <-chan struct{}) {
	glog.V(2).Infof("Pool Controller starting...")
	go c.poolController.Run(ctx)
	wait.Poll(time.Second, 5*time.Minute, func() (bool, error) {
		return c.poolController.HasSynced(), nil
	})
	if !c.poolController.HasSynced() {
		glog.Errorf("pool informer initial sync timeout")
		os.Exit(1)
	}
}

func (c *PoolController)onPoolAdd(obj interface{}) {
	glog.V(3).Infof("Add-Pool: %v", obj)
}

func (c *PoolController)onPoolUpdate(oldObj, newObj interface{}) {
	glog.V(3).Infof("Update-Pool: %v -> %v", oldObj, newObj)
}

func (c *PoolController)onPoolDel(obj interface{}) {
	glog.V(3).Infof("Del-Pool: %v", obj)
}