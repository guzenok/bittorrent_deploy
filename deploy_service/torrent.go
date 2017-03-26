// torrent
package main

import (
	"io/ioutil"
	"log"
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

func (tc *TorrentClient) SetPeers(peers []PeerInfo) {
	pp := make([]torrent.Peer, len(peers))
	for i, peerinfo := range peers {
		pp[i] = torrent.Peer{
			IP:   peerinfo.IP,
			Port: TORRENT_PORT,
		}
	}
	tc.Peers = pp
}

func (tc *TorrentClient) StartDownloadFile(fileinfo []byte) {
	var mi *metainfo.MetaInfo
	err := bencode.Unmarshal(fileinfo, mi)
	if err != nil {
		log.Printf("Decode metainfo err: %s", err.Error())
		return
	}
	t, created, _ := tc.torrentClient.AddTorrentSpec(
		torrent.TorrentSpecFromMetaInfo(mi))
	if created {
		log.Printf("Add new torrent: %v", t)
	} else {
		log.Printf("Add existing torrent: %v", t)
		return
	}
	// Ставим на закачку
	t.AddPeers(tc.Peers)
	t.DownloadAll()
}

func (tc *TorrentClient) GetFileList() map[string][]byte {
	files, err := ioutil.ReadDir(*DIR_STORE)
	if err != nil {
		log.Printf("Read local dir: %s", err.Error())
		return nil
	}
	list := make(map[string][]byte, len(files))
	for _, f := range files {
		if !f.IsDir() && f.Size() > 0 && f.Name()[0] != "."[0] {
			fileName := f.Name()
			log.Printf("Begin try seeding: %s", fileName)
			mi, err := CreateMetainfo(fileName)
			if err != nil {
				log.Printf("Create metainfo err: %s", err.Error())
				continue
			}
			_, isNew, err := tc.torrentClient.AddTorrentSpec(
				torrent.TorrentSpecFromMetaInfo(mi))
			if err != nil {
				log.Printf("Add torrent spec err: %s", err.Error())
				continue
			} else {
				log.Print("Add torrent spec OK")
			}
			if isNew {
				list[f.Name()], _ = bencode.Marshal(mi)
			}
		}
	}
	return list
}

func CreateMetainfo(fileName string) (mi *metainfo.MetaInfo, err error) {
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
