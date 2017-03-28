// consul
package main

import (
	"math/rand"
	"net"
	"strconv"

	"github.com/golang/glog"
	"github.com/hashicorp/consul/api"
)

const (
	// префикс имени сервиса в consul
	SERVICE_NAME       = "deploy"
	SERVICE_NAME_DELIM = "_"
	// префикс для consul KV
	LIST_PREFIX = SERVICE_NAME + ":file:"
	// не нужно связываться со ВСЕМИ узлами, достаточно нескольких
	PEERS_LIMIT = 10
)

type ConsulClient struct {
	NodeName      string
	AdvertiseAddr string
	config        *api.Config
	client        *api.Client
	service       *api.AgentService
	wOpt          *api.WriteOptions
	qOpt          *api.QueryOptions
}

func NewConsulClient(id string) (cc *ConsulClient) {
	cc = &ConsulClient{
		config: api.DefaultConfig(),
		service: &api.AgentService{
			ID:      SERVICE_NAME + SERVICE_NAME_DELIM + id,
			Service: SERVICE_NAME,
			Port:    TORRENT_PORT,
		},
		wOpt: &api.WriteOptions{},
		qOpt: &api.QueryOptions{},
	}
	return cc
}

func (cc *ConsulClient) hasClient() bool {
	if cc.client == nil {
		client, err := api.NewClient(cc.config)
		if err != nil {
			glog.Errorf("Consul hasClient err: %s", err.Error())
			return false
		}
		cc.client = client
	}
	return true
}

func (cc *ConsulClient) hasAgent() bool {
	if !cc.hasClient() {
		return false
	}
	agent := cc.client.Agent()
	if agent == nil {
		glog.Error("Consul has't Agent!")
		cc.needReconnect()
		return false
	}
	info, err := agent.Self()
	if err != nil {
		glog.Errorf("Consul Agent.Self err: %s!", err.Error())
		cc.needReconnect()
		return false
	}
	cc.NodeName = info["Config"]["NodeName"].(string)
	cc.AdvertiseAddr = info["Config"]["AdvertiseAddr"].(string)
	return true
}

func (cc *ConsulClient) hasKV() bool {
	if !cc.hasAgent() {
		return false
	}
	KV := cc.client.KV()
	if KV == nil {
		cc.client = nil
		glog.Error("Consul has't KV!")
		return false
	}
	return true
}

func (cc *ConsulClient) hasCatalog() bool {
	if !cc.hasAgent() {
		return false
	}
	catalog := cc.client.Catalog()
	if catalog == nil {
		cc.needReconnect()
		glog.Error("Consul has't Catalog!")
		return false
	}
	return true
}

func (cc *ConsulClient) needReconnect() {
	cc.client = nil
}

func (cc *ConsulClient) GetAnnoncedFiles() map[string][]byte {
	if !cc.hasKV() {
		return nil
	}
	pairs, _, err := cc.client.KV().List(LIST_PREFIX, cc.qOpt)
	if err != nil {
		cc.needReconnect()
		glog.Errorf("Get KV from consul err: %s!", err.Error())
		return nil
	}
	list := make(map[string][]byte, len(pairs))
	for _, pair := range pairs {
		fileName := pair.Key[len(LIST_PREFIX):]
		list[fileName] = pair.Value
	}
	return list
}

func (cc *ConsulClient) AddAnnoncedFile(key string, val *[]byte) bool {
	if !cc.hasKV() {
		return false
	}
	pair := &api.KVPair{
		Key:   LIST_PREFIX + key,
		Value: *val,
	}
	_, _, err := cc.client.KV().CAS(pair, cc.wOpt)
	if err != nil {
		cc.needReconnect()
		glog.Errorf("Add KV to consul err: %s!", err.Error())
		return false
	}
	return true
}

func (cc *ConsulClient) GetPeers() []net.IP {
	if !cc.hasCatalog() {
		return nil
	}
	services, _, err := cc.client.Catalog().Service(SERVICE_NAME, "", cc.qOpt)
	if err != nil {
		cc.needReconnect()
		glog.Errorf("Get Services from consul err: %s!", err.Error())
		return nil
	}
	// не нужно связываться со ВСЕМИ узлами, достаточно нескольких
	peersLen := len(services)
	if peersLen < 1 {
		return nil
	}
	if peersLen > PEERS_LIMIT {
		peersLen = PEERS_LIMIT
	}
	list := make([]net.IP, peersLen)
	i := 0
	registered := false
	for _, serv := range services {
		// самого себя не считаем
		if serv.Address == cc.AdvertiseAddr {
			registered = true
			continue
		}
		// поначалу всех берем, а когда уже набрали - кидаем монетку
		if i < peersLen {
			list[i] = net.ParseIP(serv.Address)
		} else if rand.Intn(100) >= 50 {
			list[i%peersLen] = net.ParseIP(serv.Address)
		}
		i++
	}
	if !registered {
		cc.Register()
	}
	return list[:peersLen]
}

func (cc *ConsulClient) Register() bool {
	return cc.registerService() && cc.registerHealthCheck()
}

func (cc *ConsulClient) registerService() bool {
	if !cc.hasCatalog() {
		return false
	}
	reg := &api.CatalogRegistration{
		Node:    cc.NodeName,
		Address: cc.AdvertiseAddr,
		Service: cc.service,
		Check: &api.AgentCheck{
			Node:      cc.NodeName,
			CheckID:   SERVICE_NAME + SERVICE_NAME_DELIM + "check",
			Name:      "Deploy health check",
			Notes:     "torrent client status",
			ServiceID: cc.service.ID,
		},
	}
	// Service
	_, err := cc.client.Catalog().Register(reg, nil)
	if err != nil {
		cc.needReconnect()
		glog.Errorf("Register service err: %s!", err.Error())
		return false
	} else {
		glog.Infof("Register service OK: %#v, %#v, %#v", *reg, *reg.Check, cc.service)
	}
	return true
}

func (cc *ConsulClient) registerHealthCheck() bool {
	if !cc.hasAgent() {
		return false
	}
	// Health check
	check := api.AgentCheckRegistration{
		// ID:        SERVICE_NAME + SERVICE_NAME_DELIM + "check",
		Name:      "Deploy health check",
		Notes:     "torrent client status",
		ServiceID: cc.service.ID,
		AgentServiceCheck: api.AgentServiceCheck{
			HTTP:     "http://127.0.0.1:" + strconv.Itoa(HEALTH_CHECK_PORT),
			Interval: "30s",
			Timeout:  "10s",
		},
	}
	err := cc.client.Agent().CheckRegister(&check)
	if err != nil {
		cc.needReconnect()
		glog.Errorf("Register healthcheck err: %s!", err.Error())
		return false
	} else {
		glog.Infof("Register healthcheck OK: %v", check)
	}
	return true
}

func (cc *ConsulClient) DeRegister() bool {
	if cc.client == nil || cc.client.Catalog() == nil {
		return false
	}
	dereg := &api.CatalogDeregistration{
		Node:      cc.NodeName,
		ServiceID: cc.service.ID,
	}
	_, err := cc.client.Catalog().Deregister(dereg, cc.wOpt)
	if err != nil {
		cc.needReconnect()
		glog.Errorf("DeRegister service err: %s", err.Error())
		return false
	} else {
		glog.Infof("DeRegister service OK: %v", *dereg)
	}
	return true
}
