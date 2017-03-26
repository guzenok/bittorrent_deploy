// consul
package main

import (
	"log"
	"strconv"

	"github.com/hashicorp/consul/api"
)

const (
	SERVICE_NAME       = "cd"
	SERVICE_NAME_DELIM = "_"
	LIST_PREFIX        = SERVICE_NAME + ":file:"
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
			log.Printf("Consul hasClient err: %s!", err.Error())
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
		log.Print("Consul has't Agent!")
		cc.needReconnect()
		return false
	}
	info, err := agent.Self()
	if err != nil {
		log.Printf("Consul Agent.Self err: %s!", err.Error())
		cc.needReconnect()
		return false
	}
	cc.NodeName = info["Config"]["NodeName"].(string)
	cc.AdvertiseAddr = info["Config"]["AdvertiseAddr"].(string)
	cc.registerService()
	// cc.registerHealthCheck()
	return true
}

func (cc *ConsulClient) hasKV() bool {
	if !cc.hasAgent() {
		return false
	}
	KV := cc.client.KV()
	if KV == nil {
		cc.client = nil
		log.Print("Consul has't KV!")
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
		log.Print("Consul has't Catalog!")
		return false
	}
	return true
}

func (cc *ConsulClient) needReconnect() {
	cc.client = nil
}

func (cc *ConsulClient) GetFileList() map[string][]byte {
	if !cc.hasKV() {
		return nil
	}
	pairs, _, err := cc.client.KV().List(LIST_PREFIX, cc.qOpt)
	if err != nil {
		cc.needReconnect()
		log.Printf("Get KV from consul err: %s!", err.Error())
		return nil
	}
	list := make(map[string][]byte, len(pairs))
	for _, pair := range pairs {
		fileName := pair.Key[len(LIST_PREFIX):]
		list[fileName] = pair.Value
	}
	return list
}

func (cc *ConsulClient) AddFileToList(key string, val []byte) bool {
	if !cc.hasKV() {
		return false
	}
	pair := &api.KVPair{
		Key:   LIST_PREFIX + key,
		Value: val,
	}
	_, _, err := cc.client.KV().CAS(pair, cc.wOpt)
	if err != nil {
		cc.needReconnect()
		log.Printf("Add KV to consul err: %s!", err.Error())
		return false
	}
	return true
}

func (cc *ConsulClient) GetPeers() []PeerInfo {
	if !cc.hasCatalog() {
		return nil
	}
	services, _, err := cc.client.Catalog().Service(SERVICE_NAME, "", cc.qOpt)
	if err != nil {
		cc.needReconnect()
		log.Printf("Get Services from consul err: %s!", err.Error())
		return nil
	}
	list := make([]PeerInfo, len(services))
	i := 0
	registered := false
	for _, serv := range services {
		if serv.Address != cc.AdvertiseAddr {
			list[i] = NewPeerInfo(serv.Address, serv.ServiceID)
			i++
		} else {
			registered = true
		}
	}
	if !registered {
		cc.needReconnect()
	}
	return list[:i]
}

func (cc *ConsulClient) registerService() bool {
	reg := &api.CatalogRegistration{
		Node:    cc.NodeName,
		Address: cc.AdvertiseAddr,
		Service: cc.service,
		/* Check: &api.AgentCheck{
			Node:      cc.NodeName,
			CheckID:   "main",
			Name:      "Deploy health check",
			Notes:     "torrent client status",
			ServiceID: cc.service.ID,
		}, */
	}
	// Service
	_, err := cc.client.Catalog().Register(reg, nil)
	if err != nil {
		cc.needReconnect()
		log.Printf("Register service err: %s!", err.Error())
		return false
	} else {
		log.Printf("Register service OK: %v, %v", *reg, reg.Service)
	}
	return true
}

func (cc *ConsulClient) registerHealthCheck() bool {
	// Health check
	check := api.AgentCheckRegistration{
		ID:        "main",
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
		log.Printf("Register healthcheck err: %s!", err.Error())
		return false
	} else {
		log.Printf("Register healthcheck OK: %v", check)
	}
	return true
}

func (cc *ConsulClient) DeRegister() bool {
	if !cc.hasCatalog() {
		return false
	}
	dereg := &api.CatalogDeregistration{
		Node:      cc.NodeName,
		ServiceID: cc.service.ID,
	}
	_, err := cc.client.Catalog().Deregister(dereg, cc.wOpt)
	if err != nil {
		cc.needReconnect()
		log.Printf("DeRegister service err: %s", err.Error())
		return false
	}
	return true
}
