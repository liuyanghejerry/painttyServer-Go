package main

import (
	"Config"
	"RoomManager"
	"log"
	"runtime"
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
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	Config.InitConf()
	var manager = RoomManager.ServeManager()
	log.Fatalln(manager.Run())
	return
}
