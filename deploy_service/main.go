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
		log.Printf("GoAgain err: %s", err.Error())
		// Listen on a TCP or a UNIX domain socket (TCP here).
		l, err = net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(HEALTH_CHECK_PORT))
		if nil != err {
			log.Fatalln(err)
		}
		log.Println("listening on", l.Addr())
		// Accept connections in a new goroutine.
	} else {
		// Resume accepting connections in a new goroutine.
		log.Println("resuming listening on", l.Addr())
		// Kill the parent, now that the child has started successfully.
		if err := goagain.Kill(); err != nil {
			log.Fatalln(err)
		}
	}

	// Запуск фоновых процессов
	stopAll := DoAll(l)

	// Block the main goroutine awaiting signals.
	if _, err := goagain.Wait(l); err != nil {
		log.Fatalln(err)
	}
	// Остановка фоновых процессов
	stopAll()
	// Do whatever's necessary to ensure a graceful exit like waiting for
	// goroutines to terminate or a channel to become closed.
	if err := l.Close(); nil != err {
		log.Fatalln(err)
	}
}
