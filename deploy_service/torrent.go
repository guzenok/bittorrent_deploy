// torrent
package main

import (
	"io/ioutil"
	"log"
	"path"
	"strconv"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/bencode"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/anacrolix/torrent/storage"
)

const (
	TORRENT_PORT = 9013
)

type TorrentClient struct {
	config        *torrent.Config
	torrentClient *torrent.Client
	Torrents      map[string]torrent.Torrent
}

func NewTorrentClient() *TorrentClient {
	var err error
	tc := &TorrentClient{}
	tc.config = &torrent.Config{
		ListenAddr:      "0.0.0.0:" + strconv.Itoa(TORRENT_PORT),
		NoDHT:           true,
		DisableTrackers: true,
		DisablePEX:      true,
		DataDir:         *DIR_STORE,
		DefaultStorage:  storage.NewFile(*DIR_CACHE),
		Debug:           true,
	}
	tc.torrentClient, err = torrent.NewClient(tc.config)
	//tc.torrentClient.
	if err != nil {
		log.Printf("Create torrent-client: %s", err.Error())
		return nil
	}
	return tc
}

func (tc *TorrentClient) StartDownloadFile(hash []byte, peers []PeerInfo) {
	if len(peers) < 1 {
		return
	}
	var Hash metainfo.Hash
	copy(Hash[:], hash[0:20])
	t, created := tc.torrentClient.AddTorrentInfoHash(Hash)
	if created {
		log.Printf("New torrent: %v", t)
	} else {
		log.Printf("Existing torrent: %v", t)
	}
	pp := make([]torrent.Peer, len(peers))
	for i, peerinfo := range peers {
		pp[i] = torrent.Peer{
			IP:   peerinfo.IP,
			Port: TORRENT_PORT,
			Id:   peerinfo.GetId(),
		}
	}
	log.Printf("Adding peers: %v", pp)
	t.AddPeers(pp)
	<-t.GotInfo()
	log.Printf("Start download pieces: %v", t.Info().Pieces)
	t.DownloadAll()
}

func (tc *TorrentClient) GetFileList(self PeerInfo) map[string][]byte {
	files, err := ioutil.ReadDir(*DIR_STORE)
	if err != nil {
		log.Printf("Read local dir: %s", err.Error())
		return nil
	}
	list := make(map[string][]byte, len(files))
	for _, f := range files {
		if !f.IsDir() && f.Size() > 0 {
			log.Printf("Begin try seeding: %s", f.Name())
			mi, err := CreateMetainfo(path.Join(*DIR_STORE, f.Name()))
			if err != nil {
				log.Printf("Create metainfo err: %s", err.Error())
				continue
			} else {
				log.Printf("Create metainfo OK: %v", mi)
			}
			list[f.Name()] = mi.HashInfoBytes().Bytes()
			t, err := tc.torrentClient.AddTorrent(mi)
			if err != nil {
				log.Printf("Add torrent err: %s", err.Error())
				continue
			} else {
				log.Printf("Add torrent OK: %v", t)
			}
			pp := []torrent.Peer{
				torrent.Peer{
					IP:   self.IP,
					Port: TORRENT_PORT,
					Id:   self.GetId(),
				},
			}
			t.AddPeers(pp)
			<-t.GotInfo()
			t.DownloadAll()
		}
	}
	return list
}

func CreateMetainfo(filePath string) (*metainfo.MetaInfo, error) {
	mi := &metainfo.MetaInfo{}
	mi.SetDefaults()
	mi.Comment = ""
	info := metainfo.Info{
		PieceLength: 256 * 1024,
	}
	err := info.BuildFromFilePath(filePath)
	if err != nil {
		return nil, err
	}
	mi.InfoBytes, err = bencode.Marshal(info)
	return mi, err
}
