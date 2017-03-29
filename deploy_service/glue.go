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
	tc *TorrentClient
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
	var (
		cc             *ConsulClient
		ServiceID      string
		processedFiles = make(map[string]*torrent.Torrent) // Список файлов в работе
	)

	// Счетчик работы
	waitCounter.Add(1)
	defer waitCounter.Done()

	// Разрегистрация в consul по окончанию
	defer func() {
		if tc != nil {
			tc.Close()
		}
		if cc != nil {
			cc.DeRegister()
		}
	}()

	// проверка отмены контекста
	isBreak := func() bool {
		select {
		case <-ctx.Done():
			return true
		default:
			return false
		}
	}

	// begin Рабочий цикл
	for {
		if isBreak() {
			return
		}
		// освобождение cpu
		time.Sleep(time.Second * 2)

		// Нужно ли подключиться к torrent?
		if tc == nil {
			// torrent-клиент
			tc = NewTorrentClient()
			if tc == nil {
				continue
			}
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

		if isBreak() {
			return
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

		if isBreak() {
			return
		}
		// Получаем адреса других пиров
		peers := cc.GetAllPeers()
		//peers := cc.GetSomePeers()
		tc.SetPeers(peers)

		if isBreak() {
			return
		}
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
				break
			}
			glog.Fatalf("Accept err: %s", err.Error())
		}
		fmt.Fprintln(c, "HTTP/1.1 200 OK")
		fmt.Fprintln(c, "Content-Type: text/plain")
		fmt.Fprintln(c, "")
		fmt.Fprintln(c, "<html><body><pre>")
		if tc != nil {
			tc.torrentClient.WriteStatus(c)
		} else {
			fmt.Fprintln(c, "nil")
		}
		fmt.Fprintln(c, "</pre></body></html>")
		c.Close() // nolint: errcheck
	}
	// end Рабочий цикл
}
