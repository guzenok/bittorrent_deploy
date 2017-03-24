// main
package main

import (
	"flag"
	"log"
	"net"
	"strconv"
	"time"

	"github.com/rcrowley/goagain"
)

const (
	HEALTH_CHECK_PORT = 80
)

var (
	DIR_STORE = flag.String("data", "./storage", "name of data dir")
	DIR_CACHE = flag.String("cache", "./storage/cache", "name of cache dir")

	tc              *TorrentClient
	cc              *ConsulClient
	goTorrentsAgain = true
	ServiceID       string
	thisPeer        PeerInfo
)

func main() {
	log.Print("Started!")
	flag.Parse()
	// Inherit a net.Listener from our parent process or listen anew.
	l, err := goagain.Listener()
	if err != nil {
		// Listen on a TCP or a UNIX domain socket (TCP here).
		l, err = net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(HEALTH_CHECK_PORT))
		if nil != err {
			log.Fatalln(err)
		}
		log.Println("listening on", l.Addr())
		// Accept connections in a new goroutine.
		go GoTorrents()
		go GoHealthChecks(l)
	} else {
		// Resume accepting connections in a new goroutine.
		log.Println("resuming listening on", l.Addr())
		go GoTorrents()
		go GoHealthChecks(l)
		// Kill the parent, now that the child has started successfully.
		if err := goagain.Kill(); err != nil {
			log.Fatalln(err)
		}
	}
	// Block the main goroutine awaiting signals.
	if _, err := goagain.Wait(l); err != nil {
		log.Fatalln(err)
	}
	// Do whatever's necessary to ensure a graceful exit like waiting for
	// goroutines to terminate or a channel to become closed.
	if err := l.Close(); nil != err {
		log.Fatalln(err)
	}
	// Разрегистрация в consul
	if cc != nil {
		cc.DeRegister()
	}
	// Остановка обработки торрентов
	goTorrentsAgain = false
	/*
		i := 3 // Ждем 3 секунды
		for tc != nil && !tc.torrentClient.WaitAll() && i > 0 {
			time.Sleep(1e9)
			i--
		}
	*/
}

// Torrents goroutine
func GoTorrents() {
	// бесконечная работа
	for goTorrentsAgain {
		// ограничение
		time.Sleep(2e9)
		// Нужно ли подключиться к torrent?
		if tc == nil {
			// torrent-клиент
			tc = NewTorrentClient()
			if tc == nil {
				continue
			}
			ServiceID = IdToString(tc.torrentClient.PeerID())
			log.Printf("PeerID: %s\n", ServiceID)
		}
		// Можно начинать работу с consul ?
		if tc != nil && cc == nil {
			// consul-клиент
			cc = NewConsulClient(ServiceID)
			// Параметры этого торрент-клиента
			thisPeer = NewPeerInfo("127.0.0.1", cc.service.ID)
		}
		// Получам списки файлов
		filesLocal := tc.GetFileList(thisPeer)
		filesAll := cc.GetFileList()

		if filesLocal != nil && filesAll != nil {
			// Опубликовать те локальные файлы, которых еще нет в списке
			for fn, hash := range filesLocal {
				if _, exist := filesAll[fn]; !exist {
					cc.AddFileToList(fn, hash)
					log.Printf("AddFileToList: %s", fn)
				}
			}
			// Поставить на закачку опубликованные файлы, которых нет локально
			peers := cc.GetPeers()
			if peers != nil && len(peers) > 0 {
				for fn, hash := range filesAll {
					if _, exist := filesLocal[fn]; !exist {
						log.Printf("StartDownloadFile %s from %v", fn, peers)
						tc.StartDownloadFile(hash, peers)
					}
				}
			}
		}
	}
}

// Health-check goroutine
func GoHealthChecks(l net.Listener) {
	for {
		c, err := l.Accept()
		if err != nil {
			if goagain.IsErrClosing(err) {
				break
			}
			log.Fatalln(err)
		}
		if tc != nil {
			tc.torrentClient.WriteStatus(c)
		} else {
			c.Write([]byte("nil"))
		}
		c.Close()
	}
}
