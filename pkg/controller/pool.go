package controller

import (
	"time"
	"os"
	"reflect"
	"strings"
	
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

type PoolController struct {
	crdClient		*rest.RESTClient
	crdScheme		*runtime.Scheme
	client			kubernetes.Interface
	
	poolController	cache.Controller
	driver			driver.GwProvider
}

func NewPoolController(client kubernetes.Interface, crdClient *rest.RESTClient, 
					crdScheme *runtime.Scheme)(*PoolController, error) {
	poolctr := &PoolController{
		crdClient 	: crdClient,
		crdScheme 	: crdScheme,
		client		: client,
	}
	driver, _ := driver.New("f5")
	poolctr.driver = driver	
	
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
	
	pool := obj.(*crdv1.ExternalNatPool)
	glog.V(3).Infof("Add-Pool: %+v", pool)
	
	poolName := utils.GeneratePoolNameEXP(pool.Namespace, pool.Name)
	err := c.driver.CreatePool(poolName, pool.Spec.Method)
	if err != nil {
		glog.Errorf("CreatePool failed: %+v\n", err)
		c.updateError(err.Error(), pool)
		return		
	}
	for _, member := range pool.Spec.Members {
		err = c.driver.AddPoolMember(poolName, member.IP, member.Port)
		glog.Errorf("AddPoolMember failed: %+v\n", err)
	}
}

func (c *PoolController)onPoolUpdate(oldObj, newObj interface{}) {
	glog.V(3).Infof("Update-Pool: %v -> %v", oldObj, newObj)
	
	if !reflect.DeepEqual(oldObj, newObj) {
		newExp := newObj.(*crdv1.ExternalNatPool)
		oldExp := oldObj.(*crdv1.ExternalNatPool)
		
		membersNew := utils.GetMembersMap(newExp)
		membersOld := utils.GetMembersMap(oldExp)
		glog.V(2).Infof("membersNew: %v", membersNew)
		glog.V(2).Infof("membersOld: %v", membersOld)
		if !reflect.DeepEqual(membersNew, membersOld) {
			glog.V(2).Infof("Need update Pool configurations.")
			poolName := utils.GeneratePoolNameEXP(oldExp.Namespace, oldExp.Name)
			c.updateExp(newExp.Namespace, poolName, membersNew, membersOld)
		}					
	}	
}

func (c *PoolController)updateExp(namespace string, poolName string, 
	membersNew map[string]int, membersOld map[string]int)error{
	for memberNew, _ := range membersNew {
		if _, ok := membersOld[memberNew]; !ok {
			glog.V(2).Infof("Pool Update: need add member %v to %s", memberNew, poolName)
			ip := strings.Split(memberNew, ":")[0]
			port := strings.Split(memberNew, ":")[1]
			err := c.driver.AddPoolMember(poolName, ip, port)
			if err != nil {
				glog.Errorf("Pool Update: add pool member failed.\n", err)
			}
		}
	}
	
	for memberOld, _ := range membersOld {
		if _, ok := membersNew[memberOld]; !ok {
			glog.V(2).Infof("Pool Update: need remove member %v from %s", memberOld, poolName)
			ip := strings.Split(memberOld, ":")[0]
			port := strings.Split(memberOld, ":")[1]
			err := c.driver.DelPoolMember(poolName, ip, port)
			if err != nil {
				glog.Errorf("Pool Update: remove pool member failed.\n", err)
			}			
		}
	}
	
	return nil
}

func (c *PoolController)onPoolDel(obj interface{}) {
	glog.V(3).Infof("Del-Pool: %v", obj)
	pool := obj.(*crdv1.ExternalNatPool)
	
	poolName := utils.GeneratePoolNameEXP(pool.Namespace, pool.Name)
	err := c.driver.DeletePool(poolName)
	if err != nil{
		glog.Errorf("DeletePool failed: %+v\n", err)
	}
}

func (c *PoolController)updateError(msg string, pool *crdv1.ExternalNatPool) {
	pool.Status.State = crdv1.POOLSTATUSERROR
	pool.Status.Message = msg
	poolclient := crdclient.PoolClient(c.crdClient, c.crdScheme, pool.Namespace)
	_, _ = poolclient.Update(pool, pool.Name)
}