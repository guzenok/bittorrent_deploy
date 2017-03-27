// glue (between ConsulClient and TorrentClient)
package main

import (
	"encoding/hex"
	"io/ioutil"
	"log"
	"time"

	"github.com/anacrolix/torrent"
)

var (
	tc              *TorrentClient
	cc              *ConsulClient
	ServiceID       string
	goTorrentsAgain = true
)

func GoTorrents() {
	// Список файлов в работе
	var processedFiles = make(map[string]*torrent.Torrent)
	// begin Рабочий цикл
	for goTorrentsAgain {

		// освобождение cpu
		time.Sleep(time.Second * 2)

		// Нужно ли подключиться к torrent?
		if tc == nil {
			// torrent-клиент
			tc = NewTorrentClient()
			if tc == nil {
				continue
			}
			defer tc.Close()
			ServiceID = hex.EncodeToString([]byte(tc.torrentClient.PeerID()))
			log.Printf("PeerID: %s\n", ServiceID)
		}

		// Можно начинать работу с consul ?
		if tc != nil && cc == nil {
			// consul-клиент
			cc = NewConsulClient(ServiceID)
			cc.Register()
		}

		// Обрабатываем имеющиеся локально файлы
		files, err := ioutil.ReadDir(*DIR_STORE)
		if err != nil {
			log.Printf("Read local dir \"%s\" err: %s", *DIR_STORE, err.Error())
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

		// Читаем все анонсы из консула
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
