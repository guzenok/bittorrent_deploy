// glue (between ConsulClient and TorrentClient)
package main

import (
	"encoding/hex"
	"net"
)

type PeerInfo struct {
	IP        net.IP
	serviceID string
}

func NewPeerInfo(addr string, servID string) PeerInfo {
	return PeerInfo{
		IP:        net.ParseIP(addr),
		serviceID: servID,
	}
}

func (this *PeerInfo) GetId() (id [20]byte) {
	prefixLen := (len(SERVICE_NAME) + len(SERVICE_NAME_DELIM))
	txt := this.serviceID[prefixLen:]
	bytes, _ := hex.DecodeString(txt)
	copy(id[:], bytes)
	return id
}

func IdToString(str string) string {
	return hex.EncodeToString([]byte(str))
}
