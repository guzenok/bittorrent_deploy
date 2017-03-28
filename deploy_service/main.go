// main
package main

import (
	"flag"
	"net"
	"strconv"

	"github.com/golang/glog"
	"github.com/rcrowley/goagain"
)

const (
	HEALTH_CHECK_PORT = 80
)

var (
	DIR_STORE = flag.String("data", "./storage", "name of data dir")
)

func main() {
	// разбор параметров
	flag.Parse()
	// логирование
	glog.Info("Started!")
	defer glog.Info("Stopped!")
	// Inherit a net.Listener from our parent process or listen anew.
	addr := "127.0.0.1:" + strconv.Itoa(HEALTH_CHECK_PORT)
	l, err := goagain.Listener()
	if err != nil {
		glog.Info("Pure start (not restart)")
		l, err = net.Listen("tcp", addr)
		if nil != err {
			glog.Fatalf("Listen %s failed: %s", addr, err.Error())
		}
		glog.Infof("Listen on %s", addr)
	} else {
		// Resume accepting connections in a new goroutine.
		glog.Infof("GoAgain try resuming listening on %s", l.Addr())
		// Kill the parent, now that the child has started successfully.
		if err := goagain.Kill(); err != nil {
			glog.Fatalf("GoAgain can't kill prevous: %s", err.Error())
		}
	}

	// Запуск фоновых процессов
	stopAll := DoAll(l)

	// Block the main goroutine awaiting signals.
	if _, err := goagain.Wait(l); err != nil {
		glog.Warningf("GoAgain killed by next: %s", err.Error())
	}

	// To ensure a graceful exit waiting for goroutines ends.
	stopAll()

	// GoAgain closing
	if err := l.Close(); nil != err {
		glog.Errorf("Closing listener err: %s", addr)
	}

}
