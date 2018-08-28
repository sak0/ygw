package controller

import (
	"time"
	"os"
	"reflect"
	"strconv"
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
	lbv1 		"github.com/sak0/ygw/pkg/apis/loadbalance/v1"
	driver 		"github.com/sak0/ygw/pkg/drivers"
	"github.com/sak0/ygw/pkg/utils"
)

type CALBPoolController struct {
	crdClient		*rest.RESTClient
	crdScheme		*runtime.Scheme
	client			kubernetes.Interface
	
	calbPoolController	cache.Controller
	driver				driver.LbProvider
}

func NewCALBPoolController(client kubernetes.Interface, crdClient *rest.RESTClient, 
					crdScheme *runtime.Scheme)(*CALBPoolController, error) {
	calbpctr := &CALBPoolController{
		crdClient 	: crdClient,
		crdScheme 	: crdScheme,
		client		: client,
	}
	driver, _ := driver.NewLBer("citrix")
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
	pool := obj.(*lbv1.CAppLoadBalancePool)
	poolName := utils.GeneratePoolNameCALBP(pool.Namespace, pool.Name)
	
	c.driver.CreatePool(poolName, pool.Spec.Method)
	var iWeight int
	for _, member := range pool.Spec.Members {
		iWeight = 1
		iPort, _ := strconv.Atoi(member.Port)
		if member.Weight != "" {
			iWeight, _ = strconv.Atoi(member.Weight)
		}
		
		c.driver.AddMemberToPool(poolName, member.IP, iPort, iWeight)
	}
}

func (c *CALBPoolController)updatePool(namespace string, poolName string, 
	membersNew map[string]int, membersOld map[string]int)error{
	for memberNew, _ := range membersNew {
		if _, ok := membersOld[memberNew]; !ok {
			glog.V(2).Infof("Pool Update: need add member %v to %s", memberNew, poolName)
			ip := strings.Split(memberNew, ":")[0]
			port := strings.Split(memberNew, ":")[1]
			iPort, _ := strconv.Atoi(port)
			weight := strings.Split(memberNew, ":")[2]
			iWeight, _ := strconv.Atoi(weight)
			err := c.driver.AddMemberToPool(poolName, ip, iPort, iWeight)
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
			iPort, _ := strconv.Atoi(port)
			err := c.driver.RemoveMemberFromPool(poolName, ip, iPort)
			if err != nil {
				glog.Errorf("Pool Update: remove pool member failed.\n", err)
			}			
		}
	}
	
	return nil
}

func (c *CALBPoolController)onPoolUpdate(oldObj, newObj interface{}) {
	glog.V(3).Infof("Update-Pool: %v -> %v", oldObj, newObj)
	if !reflect.DeepEqual(oldObj, newObj) {
		newPool := newObj.(*lbv1.CAppLoadBalancePool)
		oldPool := oldObj.(*lbv1.CAppLoadBalancePool)
		
		membersNew := utils.GetCALBMembersMap(newPool)
		membersOld := utils.GetCALBMembersMap(oldPool)
		glog.V(2).Infof("membersNew: %v", membersNew)
		glog.V(2).Infof("membersOld: %v", membersOld)
		if !reflect.DeepEqual(membersNew, membersOld) {
			glog.V(2).Infof("Need update Pool configurations.")
			poolName := utils.GeneratePoolNameCALBP(oldPool.Namespace, oldPool.Name)
			c.updatePool(newPool.Namespace, poolName, membersNew, membersOld)
		}					
	}	
}

func (c *CALBPoolController)onPoolDel(obj interface{}) {
	glog.V(3).Infof("Del-Pool: %v", obj)
	pool := obj.(*lbv1.CAppLoadBalancePool)
	poolName := utils.GeneratePoolNameCALBP(pool.Namespace, pool.Name)
	
	c.driver.DeletePool(poolName)	
}

func (c *CALBPoolController)updateError(msg string, pool *lbv1.CAppLoadBalancePool) {
	pool.Status.State = lbv1.CALBPOOLSTATUSERROR
	pool.Status.Message = msg
	poolclient := crdclient.CALBPoolClient(c.crdClient, c.crdScheme, pool.Namespace)
	_, _ = poolclient.Update(pool, pool.Name)
}