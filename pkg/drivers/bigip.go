package drivers

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"	
	
	"github.com/golang/glog"
	"github.com/scottdware/go-bigip"
	
//	"github.com/sak0/ygw/pkg/utils"	
)

const (
	F5GWPROVIDER	= "f5"
)

const iRuletmpl = `
when HTTP_REQUEST {
    set host_info [string tolower [HTTP::host]]
    switch -glob $host_info {
{{range .Rules}}
        {{.URL}} { pool {{.PoolName}} }
{{end}}      
    }
}	
`
type RuleData struct {
	Rules	[]Rule
}
type Rule struct {
	URL			string
	PoolName	string
}

type GwProvider interface {
	CreatePool(string, string)error
	AddPoolMember(string, string, string)error
	DelPoolMember(string, string, string)error
	DeletePool(string)error
	CreateVirtualServer(string, string, string, string, string)error
	DeleteVirtualServer(string)error
	VirtualServerBindPool(string, string)error
}

type F5er struct{
	client	*bigip.BigIP
}

func (f5 *F5er)createVirtualServerNat(name, ip, port, protocol string)error{
	profiles := []bigip.Profile{
		bigip.Profile{
			Name: "fastL4",
			Context: "all",		
		},
	}	
	dest := ip + ":" + port
	
	vsConfig := &bigip.VirtualServer{
		Name : name,
		Mask : "255.255.255.255",
		Destination : dest,
		IPProtocol : protocol,
		RateLimit : "10240",
		Profiles : profiles,
	}
	return f5.client.AddVirtualServer(vsConfig)
}

func (f5 *F5er)recreateVirtualServerURL(name, ip, port string)error{
	err := f5.deleteVirtualServer(name)
	if err != nil {
		return err
	}
	return f5.createVirtualServerURL(name, ip, port)
}

func (f5 *F5er)deleteVirtualServer(name string)error{
	return f5.client.DeleteVirtualServer(name)
}

func (f5 *F5er)DeleteVirtualServer(name string)error{
	return f5.deleteVirtualServer(name)
}

func (f5 *F5er)createVirtualServerURL(name, ip, port string)error{
	profiles := []bigip.Profile{
		bigip.Profile{
			Name: "http",
			Context: "all",
		},
		bigip.Profile{
			Name: "tcp",
			Context: "all",		
		},
	}
	
	dest := ip + ":" + port
	
	vsConfig := &bigip.VirtualServer{
		Name : name,
		Mask : "255.255.255.255",
		Destination : dest,
		IPProtocol : "tcp",
		RateLimit : "10240",
		Profiles: profiles,
	}
	return f5.client.AddVirtualServer(vsConfig)
}
func (f5 *F5er)CreateVirtualServer(vsType string, name string, ip string, port string, protocol string)error{
	var err error
	
	switch vsType {
		case "nat":
			err = f5.createVirtualServerNat(name, ip, port, protocol)
			if err != nil {
				if strings.Contains(err.Error(), "already exists") {
					glog.Infof("virtualServer %s Already exists, skip create.", name)
				} else {
					return err
				}					
			}
		case "url":
			err = f5.createVirtualServerURL(name, ip, port)
			if err != nil {
				if strings.Contains(err.Error(), "already exists") {
					fmt.Printf("Already exists, skip create\n")
				} else {
					return err
				}					
			}
		default:
			return fmt.Errorf("Can't support vsType %s", vsType)					
	}
	return nil
}

func (f5 *F5er)VirtualServerBindPool(vsName, poolName string)error{
	vsConfig := &bigip.VirtualServer{
		Name : vsName,
		Pool : poolName,
	}	
	return f5.client.ModifyVirtualServer(vsName, vsConfig)	
}
func (f5 *F5er)VirtualServerUnbindPool(vsName, poolName string)error{
	vsConfig := &bigip.VirtualServer{
		Name : vsName,
		Pool : "None",
	}	
	return f5.client.ModifyVirtualServer(vsName, vsConfig)	
}
func (f5 *F5er)VirtualServerBindURL(vsName, URL, poolName string)error{
    vs, err := f5.client.GetVirtualServer(vsName)
    if err != nil || vs == nil {
    	glog.Errorf("GetVirtualServer %s failed.", vsName)
		return err   
    }
    rules := vs.Rules	
	
	iRuleName := "iRule_" + vsName + "_" + URL + "_" + poolName
	buff := bytes.NewBufferString("")
	ruleTmpl := template.Must(template.New("irule").Parse(iRuletmpl))
	data := RuleData{
		Rules : []Rule{
			Rule{
				URL : URL,
				PoolName : poolName,
			},		
		},
	}
	ruleTmpl.Execute(buff, data)
	
    err = f5.client.CreateIRule(iRuleName, buff.String())
    if err != nil {
	    if strings.Contains(err.Error(), "already exists") {
		    glog.Infof("iRule %s Already exists. skip create.", iRuleName)
	    } else {
		    return err
	    }
    }
    rules = append(rules, iRuleName)

	vsConfig := &bigip.VirtualServer{
		Name : vsName,
		Rules : rules,
	}	
	return f5.client.ModifyVirtualServer(vsName, vsConfig) 
}

func (f5 *F5er)VirtualServerUnbindURL(vsName, URL, poolName string)error{
    vs, err := f5.client.GetVirtualServer(vsName)
    if err != nil {
		return err   
    }
    addr := vs.Destination
    ip := strings.Split(addr, ":")[0]
    port := strings.Split(addr, ":")[1]
    rules := vs.Rules
    
    iRuleName := "iRule_" + vsName + "_" + URL + "_" + poolName
    index := -1
    for i, rule := range rules {
    	ruleName := strings.Split(rule, "/")[2]
	    if ruleName == iRuleName {
		    index = i
	    }
    }
    if index == -1 {
    	glog.Infof("rule %s is not associate with virtual server.", iRuleName)
//	    return nil
    } else {
	    rules = append(rules[:index], rules[index + 1:]...)
	    if len(rules) == 0 {
	    	//TODO : no rest api for clean ruls yet. workaround for Recreate
	    	glog.Infof("Recreate VirtualServer %s for clean iRules.", vsName) 
		    return f5.recreateVirtualServerURL(vsName, ip, port)
	    }
    
	 	vsConfig := &bigip.VirtualServer{
			Name : vsName,
			Rules : rules,
		}	
		err = f5.client.ModifyVirtualServer(vsName, vsConfig)
		if err != nil {
			glog.Errorf("configure virtual server failed: %v\n", err)
		}    
    }
    
	glog.Infof("Delete iRule: %s", iRuleName)
	err = f5.client.DeleteIRule(iRuleName)
	if err != nil {
		if strings.Contains(err.Error(), "was not found") {
			glog.Warningf("iRule %s is not exists.", iRuleName)
		} else {
			return err
		}
	}
	
	return nil     	
}

func (f5 *F5er)CreatePool(poolName, lbMethod string)error{
	err := f5.client.CreatePool(poolName)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			glog.Infof("pool %s Already exists, skip create.", poolName)
		} else {
			return err
		}		
	}	
	
	poolConfig := &bigip.Pool{
		Name : poolName,
		LoadBalancingMode : lbMethod,
	}
	return f5.client.ModifyPool(poolName, poolConfig)
}

func (f5 *F5er)DeletePool(poolName string)error{
	err := f5.client.DeletePool(poolName)
	if err != nil {
		if strings.Contains(err.Error(), "was not found") {
			glog.Warningf("Pool %s is not exists.", poolName)
		} else {
			return err
		}	
	}
	
	return nil
}

func (f5 *F5er)AddPoolMember(poolName, memberIp, memberPort string)error{
	memberConfig := &bigip.PoolMember {
		Name : memberIp + ":" + memberPort,
	}
	
	err := f5.client.CreateNode(memberIp, memberIp)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			glog.Infof("node %s Already exists, skip create.", memberIp)
		} else {
			return err
		}		
	}	
	
	err = f5.client.CreatePoolMember(poolName, memberConfig)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			glog.Infof("poolmember %s Already exists, skip create.", memberConfig.Name)
		} else {
			return err
		}		
	}
	
	return nil	
}

func (f5 *F5er)DelPoolMember(poolName, memberIp, memberPort string)error{
	member := memberIp + ":" + memberPort
	err := f5.client.DeletePoolMember(poolName, member)
	if err != nil {
		if strings.Contains(err.Error(), "was not found") {
			glog.Infof("Already deleted, skip delete.")
		} else {
			return err
		}		
	}
	
	return f5.client.DeleteNode(memberIp)
}

func New(gwtype string)(GwProvider, error){
	switch gwtype {
		case F5GWPROVIDER:
			client := bigip.NewSession("10.0.10.253", "user", "yhcs_user", nil)
			f5er := &F5er{
				client : client,
			}			
			return f5er, nil
		default:
			return nil, fmt.Errorf("Unsupport type: %s", gwtype)
	}
}