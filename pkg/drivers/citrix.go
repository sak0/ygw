package drivers

import (
	"fmt"
	"strconv"	
	
	"github.com/golang/glog"
		
	citrixbasic "github.com/chiradeep/go-nitro/config/basic"
	/*"github.com/chiradeep/go-nitro/config/cs"*/
	citrixlb "github.com/chiradeep/go-nitro/config/lb"
	"github.com/chiradeep/go-nitro/netscaler"
)

const (
	CITRIXLBPROVIDER	 = "citrix"
)

type LbProvider interface {
	CreatePool(string, string)error
	AddMemberToPool(string, string, int, int)error
	RemoveMemberFromPool(string, string, int)error
	DeletePool(string)error
}

type CitrixLb struct{}

func (c *CitrixLb)createSvcGroup(groupName string)error{
	client, _ := netscaler.NewNitroClientFromEnv()
	nsSvcGrp := citrixbasic.Servicegroup{
		Servicegroupname	: groupName,
		Servicetype			: "HTTP",
	}
	_, err := client.AddResource(netscaler.Servicegroup.Type(), groupName, &nsSvcGrp)
	if err != nil {
		return err
	}
	return nil	
}

func (c *CitrixLb)deleteSvcGroup(groupName string)error{
	glog.V(2).Infof("Citrix Driver DeleteSvcGroup")
	client, _ := netscaler.NewNitroClientFromEnv()
	err := client.DeleteResource(netscaler.Servicegroup.Type(), groupName)
	if err != nil {
		return err
	}
	return nil	
}

func (c *CitrixLb)createVs(vsName string, method string)error{
	client, err := netscaler.NewNitroClientFromEnv()
	if err != nil {
		return err
	}
	
	nsLB := citrixlb.Lbvserver{
		Name			: vsName,
		Servicetype		: "HTTP",
		//Lbmethod        : "ROUNDROBIN",
		Lbmethod        : method,
	}
	name, err := client.AddResource(netscaler.Lbvserver.Type(), vsName, &nsLB)
	if err != nil {
		glog.Errorf("Citrix create Lbvserver failed %v", err)
	} else {
		glog.V(2).Infof("Citrix created Lbvserver %s", name)
	}
		
	return nil
}

func (c *CitrixLb)deleteVs(vsName string)error{
	client, _ := netscaler.NewNitroClientFromEnv()
	err := client.DeleteResource(netscaler.Lbvserver.Type(), vsName)
	if err != nil {
		return err
	}
	return nil	
}

func (c *CitrixLb)bindSvcGroupVs(groupName, vsName string)error{
	glog.V(2).Infof("Citrix Driver BindSvcGroupLb. bind %s to %s", groupName, vsName)
	client, _ := netscaler.NewNitroClientFromEnv()
	binding := citrixlb.Lbvserverservicegroupbinding{
		Servicegroupname	: groupName,
		Name				: vsName,
	}
	err := client.BindResource(netscaler.Lbvserver.Type(), vsName, netscaler.Servicegroup.Type(), groupName, &binding)
	if err != nil {
		return err
	} 
	return nil	
}

func (c *CitrixLb)CreatePool(poolName string, method string)error {
	err := c.createSvcGroup(poolName)
	if err != nil {
		return err
	}
	
	err = c.createVs(poolName, method)
	if err != nil {
		return err
	}
	
	err = c.bindSvcGroupVs(poolName, poolName)
	if err != nil {
		return err
	}	
	
	return nil	
}

func (c *CitrixLb)DeletePool(poolName string)error {
	err := c.deleteVs(poolName)
	if err != nil {
		return err
	}
	
	err = c.deleteSvcGroup(poolName)
	if err != nil {
		return err
	}
	
	return nil	
}

func (c *CitrixLb)createServer(ip string)error{
	client, _ := netscaler.NewNitroClientFromEnv()
	nsServer := citrixbasic.Server{
		Name			: ip,
		Ipaddress		: ip,
	}
	_, err := client.AddResource(netscaler.Server.Type(), ip, &nsServer)
	if err != nil {
		glog.Errorf("createServer failed: %v", err)
	}	
	return nil	
}

func (c *CitrixLb)bindServerToGroup(groupName string, serverName string, port, weight int)error{
	glog.V(2).Infof("Citrix Driver BindServerToGroup %s->%s", serverName, groupName)
	
	client, _ := netscaler.NewNitroClientFromEnv()
	binding := citrixbasic.Servicegroupservicegroupmemberbinding{
		Servicegroupname	: groupName,
		Servername			: serverName,
		Port				: port,
		Weight				: weight,
	}
	//err := client.BindResource(netscaler.Servicegroup.Type(), groupName, netscaler.Server.Type(), serverName, &binding)
	_, err := client.AddResource(netscaler.Servicegroup_servicegroupmember_binding.Type(), groupName, &binding)
	if err != nil {
		return err
	} 	
	
	return nil	
}

func (c *CitrixLb)AddMemberToPool(groupName string, ip string, port, weight int)error{
	err := c.createServer(ip)
	if err != nil {
		return err
	}
	
	err = c.bindServerToGroup(groupName, ip, port, weight)
	if err != nil {
		return err
	}
	
	return nil
}

func (c *CitrixLb)unbindServerToGroup(groupName, serverName string, port int)error{
	glog.V(2).Infof("Citrix Driver UnBindServerFromGroup %s->%s", serverName, groupName)

	client, _ := netscaler.NewNitroClientFromEnv()
	var args = []string{
		"servername:" + serverName,
		"servicegroupname:" + groupName,
		"port:" + strconv.Itoa(port),
	}
	
	err := client.DeleteResourceWithArgs(netscaler.Servicegroup_servicegroupmember_binding.Type(), groupName, args)
	if err != nil {
		return err
	}
	return nil
}

func (c *CitrixLb)RemoveMemberFromPool(groupName, serverName string, port int)error{
	return c.unbindServerToGroup(groupName, serverName, port)
}

func NewLBer(lbtype string)(LbProvider, error){
	switch lbtype {
		case CITRIXLBPROVIDER:
			return &CitrixLb{}, nil
		default:
			return nil, fmt.Errorf("Unsupport type: %s", lbtype)
	}
}