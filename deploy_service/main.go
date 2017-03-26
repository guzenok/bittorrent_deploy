// main
package main

import (
	"flag"
	"log"
	"net"
	"strconv"

	"github.com/rcrowley/goagain"
)

const (
	HEALTH_CHECK_PORT = 80
)

var (
	DIR_STORE = flag.String("data", "./storage", "name of data dir")
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
			time.Sleep(time.Second * 1)
			i--
		}
	*/
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
			c.Write([]byte("nil")) // nolint: errcheck
		}
		c.Close() // nolint: errcheck
	}
}
