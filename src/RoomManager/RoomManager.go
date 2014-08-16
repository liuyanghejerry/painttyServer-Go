package RoomManager

import (
	//"log"
	"net"
	//"time"
)

func ServeManager() {
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		// handle error
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			// handle error
			continue
		}
		//go handleConnection(conn)
		_= conn
	}
}
