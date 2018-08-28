package drivers

import (
	"fmt"
	"strconv"
	"sort"
	
	"github.com/golang/glog"
	
	"github.com/chiradeep/go-nitro/config/cs"	
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
	
	CreateLB(string, string, int)error
	DeleteLB(string)error
	AddRuleToLB(string, string, string, string, string, string)error
	RemoveRuleToLB(string, string, string, string, string, string)error
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
		glog.Errorf("createServer failed: %v", err)
	}
	
	err = c.bindServerToGroup(groupName, ip, port, weight)
	if err != nil {
		glog.Errorf("bindServerToGroup failed: %v", err)
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

func (c *CitrixLb)createContentVs(csvserverName string, vserverIp string, vserverPort int)error{
	client, _ := netscaler.NewNitroClientFromEnv()
	protocol := "HTTP"
	cs := cs.Csvserver{
		Name:        csvserverName,
		Ipv46:       vserverIp,
		Servicetype: protocol,
		Port:        vserverPort,
	}
	_, _ = client.AddResource(netscaler.Csvserver.Type(), csvserverName, &cs)
	return nil
}

func (c *CitrixLb)CreateLB(lbName string, vip string, port int)error{
	return c.createContentVs(lbName, vip, port)
}

func (c *CitrixLb)RemoveRuleToLB(lbName string, domainName string, path string, 
	poolName string, actionName string, policyName string)error{
	client, _ := netscaler.NewNitroClientFromEnv()
	
	err := client.UnbindResource(netscaler.Csvserver.Type(), lbName, netscaler.Cspolicy.Type(), policyName, "policyName")
	if err != nil {
		glog.Errorf("UnbindPolicy failed: %v", err)
	}		
	err = client.DeleteResource(netscaler.Cspolicy.Type(), policyName)
	if err != nil {
		glog.Errorf("DeletePolicy failed: %v", err)
	}
	
	err = client.DeleteResource(netscaler.Csaction.Type(), actionName)
	if err != nil {
		glog.Errorf("DeleteAction failed: %v", err)
	}	
	
	return nil
}
	
func (c *CitrixLb)ListBoundPolicies(csvserverName string) ([]string, []int) {
	ret1 := []string{}
	ret2 := []int{}
	client, _ := netscaler.NewNitroClientFromEnv()
	policies, err := client.FindAllBoundResources(netscaler.Csvserver.Type(), csvserverName, netscaler.Cspolicy.Type())
	if err != nil {
		glog.Errorf("No bindings for CS Vserver %s: %v", csvserverName, err)
		return ret1, ret2
	}
	for _, policy := range policies {
		pname := policy["policyname"].(string)
		prio, err := strconv.Atoi(policy["priority"].(string))
		if err != nil {
			continue
		}
		ret1 = append(ret1, pname)
		ret2 = append(ret2, prio)

	}
	sort.Ints(ret2)
	return ret1, ret2
}	

func (c *CitrixLb)AddRuleToLB(lbName string, domainName string, path string, 
	poolName string, actionName string, policyName string)error{
	var priority = 1
	_, priorities := c.ListBoundPolicies(lbName)
	if len(priorities) > 0 {
		priority = priorities[len(priorities)-1] + 1
	}		
		
	client, _ := netscaler.NewNitroClientFromEnv()	
	csAction := cs.Csaction{
		Name:            actionName,
		Targetlbvserver: poolName,
	}
	_, _ = client.AddResource(netscaler.Csaction.Type(), actionName, &csAction)
	
	var rule string
	if path != "" {
		rule = fmt.Sprintf("HTTP.REQ.HOSTNAME.EQ(\"%s\") && HTTP.REQ.URL.PATH.EQ(\"%s\")", domainName, path)
	} else {
		rule = fmt.Sprintf("HTTP.REQ.HOSTNAME.EQ(\"%s\")", domainName)
	}
	csPolicy := cs.Cspolicy{
		Policyname: policyName,
		Rule:       rule,
		Action:     actionName,
	}
	_, _ = client.AddResource(netscaler.Cspolicy.Type(), policyName, &csPolicy)

	binding2 := cs.Csvservercspolicybinding{
		Name:       lbName,
		Policyname: policyName,
		Priority:   priority,
		Bindpoint:  "REQUEST",
	}
	_ = client.BindResource(netscaler.Csvserver.Type(), lbName, netscaler.Cspolicy.Type(), policyName, &binding2)	
	return nil
}
	
func (c *CitrixLb)DeleteLB(lbName string)error{
	client, _ := netscaler.NewNitroClientFromEnv()	
	err := client.DeleteResource(netscaler.Csvserver.Type(), lbName)
	if err != nil {
		return err
	}
	return nil
}

func NewLBer(lbtype string)(LbProvider, error){
	switch lbtype {
		case CITRIXLBPROVIDER:
			return &CitrixLb{}, nil
		default:
			return nil, fmt.Errorf("Unsupport type: %s", lbtype)
	}
}