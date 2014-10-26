package main

import (
	"RoomManager"
	"errors"
	"log"
)

import _ "net/http/pprof"
import http "net/http"

func init() {
	go func() {
		log.Println(http.ListenAndServe("localhost:6767", nil))
	}()
}

func runServer() (err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Server is down:", r)
			err = errors.New("Server is down")
		}
	}()

	var manager = RoomManager.ServeManager()
	manager.Run()
	return nil
}

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	for re, times := runServer(), 0; re != nil && times < 20; times++ {
		re = runServer()
	}
	log.Println("Server died.")
	return
}
