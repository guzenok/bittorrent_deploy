// glue (between ConsulClient and TorrentClient)
package main

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net"
	"sync"
	"time"

	"github.com/golang/glog"

	"github.com/anacrolix/torrent"
	"github.com/rcrowley/goagain"
	"golang.org/x/net/context"
)

var (
	tc        *TorrentClient
	cc        *ConsulClient
	ServiceID string
)

func DoAll(l net.Listener) func() {
	var (
		waitCounter               sync.WaitGroup
		work1Context, work1Cancel = context.WithCancel(context.Background())
		work2Context, work2Cancel = context.WithCancel(context.Background())
	)
	go GoTorrents(work1Context, &waitCounter)
	go GoHealthChecks(work2Context, &waitCounter, l)
	return func() {
		work1Cancel()
		work2Cancel()
		waitCounter.Wait()
	}
}

func GoTorrents(ctx context.Context, waitCounter *sync.WaitGroup) {
	// Список файлов в работе
	var processedFiles = make(map[string]*torrent.Torrent)
	// Счетчик работы
	waitCounter.Add(1)
	defer waitCounter.Done()
	// Разрегистрация в consul
	defer func() {
		if cc != nil {
			cc.DeRegister()
		}
	}()
	// begin Рабочий цикл
	for {
		// выход по сигналу контекста
		select {
		case <-ctx.Done():
			return
		default:
			time.Sleep(time.Second * 2) // освобождение cpu
		}

		// Нужно ли подключиться к torrent?
		if tc == nil {
			// torrent-клиент
			tc = NewTorrentClient()
			if tc == nil {
				continue
			}
			defer tc.Close()
			ServiceID = hex.EncodeToString([]byte(tc.torrentClient.PeerID()))
			glog.Infof("PeerID: %s\n", ServiceID)
		}

		// Можно начинать работу с consul ?
		if tc != nil && cc == nil {
			// consul-клиент
			cc = NewConsulClient(ServiceID)
			if !cc.Register() {
				continue
			}
		}

		// Обрабатываем имеющиеся локально файлы
		files, err := ioutil.ReadDir(*DIR_STORE)
		if err != nil {
			glog.Errorf("Read local dir \"%s\" err: %s", *DIR_STORE, err.Error())
			continue
		} else {
			for _, f := range files {
				// не берем в работу директории, пустые и скрытые файлы
				if f.IsDir() || f.Size() < 1 || f.Name()[0] == "."[0] {
					continue
				}
				fileName := f.Name()
				// если еще не обработан, то
				if _, processed := processedFiles[fileName]; !processed {
					// начинаем раздачу и анонсируем ее в consul
					t, annonce := tc.Share(fileName)
					if t != nil && cc.AddAnnoncedFile(fileName, annonce) {
						processedFiles[fileName] = t
					}
				}
			}
		}

		// Получаем адреса других пиров
		peers := cc.GetPeers()
		tc.SetPeers(peers)

		// Читаем все анонсы из consul
		annoncedFiles := cc.GetAnnoncedFiles()
		if annoncedFiles == nil {
			continue
		}
		for fileName, annonce := range annoncedFiles {
			// если еще не обработан, то
			if _, processed := processedFiles[fileName]; !processed {
				// ставим на закачку
				t := tc.StartDownloadFile(fileName, annonce)
				if t != nil {
					processedFiles[fileName] = t
				}
			}
		}

	}
	// end Рабочий цикл
}

// Health-check goroutine
func GoHealthChecks(ctx context.Context, waitCounter *sync.WaitGroup, l net.Listener) {
	// Счетчик работы
	waitCounter.Add(1)
	defer waitCounter.Done()
	// begin Рабочий цикл
	for {
		// выход по сигналу контекста
		select {
		case <-ctx.Done():
			return
		default:
		}
		// Ответ на запросы
		c, err := l.Accept()
		if err != nil {
			if goagain.IsErrClosing(err) {
				return
			}
			glog.Fatalf("Accept err: %s", err.Error())
		}
		fmt.Fprintln(c, "HTTP/1.1 200 OK")
		fmt.Fprintln(c, "Content-Type: text/plain")
		fmt.Fprintln(c, "")
		if tc != nil {
			tc.torrentClient.WriteStatus(c)
		} else {
			c.Write([]byte("nil")) // nolint: errcheck
		}
		fmt.Fprintln(c, "")
		c.Close() // nolint: errcheck
	}
	// end Рабочий цикл
}
