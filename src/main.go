package main

import (
	"RoomManager"
	"log"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	var manager = RoomManager.ServeManager()
	log.Println(manager.Run())
	return
}
