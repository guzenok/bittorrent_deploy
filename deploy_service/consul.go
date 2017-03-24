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

func NewConsulClient(id string) (this *ConsulClient) {
	this = &ConsulClient{
		config: api.DefaultConfig(),
		service: &api.AgentService{
			ID:      SERVICE_NAME + SERVICE_NAME_DELIM + id,
			Service: SERVICE_NAME,
			Port:    TORRENT_PORT,
		},
		wOpt: &api.WriteOptions{},
		qOpt: &api.QueryOptions{},
	}
	return this
}

func (this *ConsulClient) hasClient() bool {
	if this.client == nil {
		client, err := api.NewClient(this.config)
		if err != nil {
			log.Printf("Consul hasClient err: %s!", err.Error())
			return false
		}
		this.client = client
	}
	return true
}

func (this *ConsulClient) hasAgent() bool {
	if !this.hasClient() {
		return false
	}
	agent := this.client.Agent()
	if agent == nil {
		log.Print("Consul has't Agent!")
		this.needReconnect()
		return false
	}
	info, err := agent.Self()
	if err != nil {
		log.Printf("Consul Agent.Self err: %s!", err.Error())
		this.needReconnect()
		return false
	}
	this.NodeName = info["Config"]["NodeName"].(string)
	this.AdvertiseAddr = info["Config"]["AdvertiseAddr"].(string)
	this.registerService()
	// this.registerHealthCheck()
	return true
}

func (this *ConsulClient) hasKV() bool {
	if !this.hasAgent() {
		return false
	}
	KV := this.client.KV()
	if KV == nil {
		this.client = nil
		log.Print("Consul has't KV!")
		return false
	}
	return true
}

func (this *ConsulClient) hasCatalog() bool {
	if !this.hasAgent() {
		return false
	}
	catalog := this.client.Catalog()
	if catalog == nil {
		this.needReconnect()
		log.Print("Consul has't Catalog!")
		return false
	}
	return true
}

func (this *ConsulClient) needReconnect() {
	this.client = nil
}

func (this *ConsulClient) GetFileList() map[string][]byte {
	if !this.hasKV() {
		return nil
	}
	pairs, _, err := this.client.KV().List(LIST_PREFIX, this.qOpt)
	if err != nil {
		this.needReconnect()
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

func (this *ConsulClient) AddFileToList(key string, val []byte) bool {
	if !this.hasKV() {
		return false
	}
	pair := &api.KVPair{
		Key:   LIST_PREFIX + key,
		Value: val,
	}
	_, _, err := this.client.KV().CAS(pair, this.wOpt)
	if err != nil {
		this.needReconnect()
		log.Printf("Add KV to consul err: %s!", err.Error())
		return false
	}
	return true
}

func (this *ConsulClient) GetPeers() []PeerInfo {
	if !this.hasCatalog() {
		return nil
	}
	services, _, err := this.client.Catalog().Service(SERVICE_NAME, "", this.qOpt)
	if err != nil {
		this.needReconnect()
		log.Printf("Get Services from consul err: %s!", err.Error())
		return nil
	}
	list := make([]PeerInfo, len(services))
	i := 0
	registered := false
	for _, serv := range services {
		if serv.Address != this.AdvertiseAddr {
			list[i] = NewPeerInfo(serv.Address, serv.ServiceID)
			i++
		} else {
			registered = true
		}
	}
	if !registered {
		this.needReconnect()
	}
	return list[:i]
}

func (this *ConsulClient) registerService() bool {
	reg := &api.CatalogRegistration{
		Node:    this.NodeName,
		Address: this.AdvertiseAddr,
		Service: this.service,
		Check: &api.AgentCheck{
			Node:      this.NodeName,
			CheckID:   "main",
			Name:      "Deploy health check",
			Notes:     "torrent client status",
			ServiceID: this.service.ID,
		},
	}
	// Service
	_, err := this.client.Catalog().Register(reg, nil)
	if err != nil {
		this.needReconnect()
		log.Printf("Register service err: %s!", err.Error())
		return false
	} else {
		log.Printf("Register service OK: %v, %v", *reg, reg.Service)
	}
	return true
}

func (this *ConsulClient) registerHealthCheck() bool {
	// Health check
	check := api.AgentCheckRegistration{
		ID:        "main",
		Name:      "Deploy health check",
		Notes:     "torrent client status",
		ServiceID: this.service.ID,
		AgentServiceCheck: api.AgentServiceCheck{
			HTTP:     "http://127.0.0.1:" + strconv.Itoa(HEALTH_CHECK_PORT),
			Interval: "30s",
			Timeout:  "10s",
		},
	}
	err := this.client.Agent().CheckRegister(&check)
	if err != nil {
		this.needReconnect()
		log.Printf("Register healthcheck err: %s!", err.Error())
		return false
	} else {
		log.Printf("Register healthcheck OK: %v", check)
	}
	return true
}

func (this *ConsulClient) DeRegister() bool {
	if !this.hasCatalog() {
		return false
	}
	dereg := &api.CatalogDeregistration{
		Node:      this.NodeName,
		ServiceID: this.service.ID,
	}
	_, err := this.client.Catalog().Deregister(dereg, this.wOpt)
	if err != nil {
		this.needReconnect()
		log.Printf("DeRegister service err: %s", err.Error())
		return false
	}
	return true
}
