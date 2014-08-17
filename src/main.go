package main

import (
	"RoomManager"
)

func main() {
	var manager = RoomManager.ServeManager()
	manager.Run()
	return
}
