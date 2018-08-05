package main

import (
	"log"
	"runtime"
	"server/pkg/Config"
	"server/pkg/Logger"
	"server/pkg/RoomManager"
	"time"
)

import _ "net/http/pprof"
import http "net/http"

func init() {
	runtime.SetBlockProfileRate(1)
	go func() {
		log.Println(http.ListenAndServe("localhost:6767", nil))
	}()
}

func main() {
	logger.SetupLogs("painttyServer")
	Config.InitConf()
	var manager = RoomManager.ServeManager()

	ticker := time.NewTicker(10 * time.Minute)
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				Config.ReloadConf()
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()

	log.Fatalln(manager.Run())

	close(quit)

	return
}
