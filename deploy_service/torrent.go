// torrent
package main

import (
	"log"
	"net"
	"path"
	"strconv"

	"github.com/anacrolix/dht"
	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/bencode"
	"github.com/anacrolix/torrent/metainfo"
)

const (
	TORRENT_PORT = 9013
)

type TorrentClient struct {
	config        *torrent.Config
	torrentClient *torrent.Client
	Torrents      map[string]torrent.Torrent
	Peers         []torrent.Peer
}

func NewTorrentClient() *TorrentClient {
	var err error
	tc := &TorrentClient{}
	tc.config = &torrent.Config{
		ListenAddr: "0.0.0.0:" + strconv.Itoa(TORRENT_PORT),
		DataDir:    *DIR_STORE,
		Seed:       true,
		DisablePEX: false,
		NoDHT:      true,
		DHTConfig: dht.ServerConfig{
			NoDefaultBootstrap: true,
		},
		DisableTrackers: true,
		Debug:           true,
	}
	tc.torrentClient, err = torrent.NewClient(tc.config)
	if err != nil {
		log.Printf("Create torrent-client err: %s", err.Error())
		return nil
	}
	return tc
}

func (tc *TorrentClient) Close() {
	if tc.torrentClient != nil {
		tc.torrentClient.Close()
	}
}

func (tc *TorrentClient) SetPeers(peers []net.IP) {
	pp := make([]torrent.Peer, len(peers))
	for i, ip := range peers {
		pp[i] = torrent.Peer{
			IP:   ip,
			Port: TORRENT_PORT,
		}
	}
	tc.Peers = pp
}

func (tc *TorrentClient) StartDownloadFile(fileName string, annonce []byte) (t *torrent.Torrent) {
	var mi metainfo.MetaInfo
	err := bencode.Unmarshal(annonce, &mi)
	if err != nil {
		log.Printf("Deserialize metainfo for \"%s\" err: %s", fileName, err.Error())
		return
	}
	t = setTorrent(&mi, "leeching", fileName)
	// Ставим на закачку
	if t != nil {
		t.AddPeers(tc.Peers)
		<-t.GotInfo()
		t.DownloadAll()
	}
	return
}

func (tc *TorrentClient) Share(fileName string) (t *torrent.Torrent, annonce *[]byte) {
	log.Printf("Try share file \"%s\"", fileName)
	mi, err := createMetainfo(fileName)
	if err != nil {
		log.Printf("Create metainfo for \"%s\" err: %s", fileName, err.Error())
		return
	}
	t = setTorrent(mi, "seeding", fileName)
	// готовим возвращаемые значения
	bytes, err := bencode.Marshal(mi)
	if err != nil {
		log.Printf("Serialize metainfo for \"%s\" err: %s", fileName, err.Error())
	}
	annonce = &bytes
	return
}

func setTorrent(mi *metainfo.MetaInfo, act string, fileName string) *torrent.Torrent {
	newT, isNew, err := tc.torrentClient.AddTorrentSpec(torrent.TorrentSpecFromMetaInfo(mi))
	if err != nil {
		log.Printf("Add torrent for \"%s\" err: %s", fileName, err.Error())
		return nil
	}
	if isNew {
		log.Printf("Begin %s \"%s\"", act, fileName)
	} else {
		log.Printf("Already %s \"%s\"", act, fileName)
	}
	return newT
}

func createMetainfo(fileName string) (mi *metainfo.MetaInfo, err error) {
	mi = nil
	// параметры файла
	filePath := path.Join(*DIR_STORE, fileName)
	info := metainfo.Info{
		PieceLength: 512 * 1024,
	}
	err = info.BuildFromFilePath(filePath)
	if err != nil {
		return
	}
	mi = &metainfo.MetaInfo{}
	mi.InfoBytes, err = bencode.Marshal(info)
	return
}
