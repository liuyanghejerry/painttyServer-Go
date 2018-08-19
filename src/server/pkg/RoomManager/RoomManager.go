package RoomManager

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	dbutil "github.com/syndtr/goleveldb/leveldb/util"
	"log"
	"net"
	"server/pkg/Config"
	"server/pkg/Room"
	"server/pkg/Hub"
	"server/pkg/Router"
	"server/pkg/Socket"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type RoomManager struct {
	ln               *net.TCPListener
	goingClose       chan bool
	router           *Router.Router
	rooms            sync.Map
	currentRoomCount int32
	db               *leveldb.DB
}

func (m *RoomManager) init() error {
	m.goingClose = make(chan bool)
	m.router = Router.MakeRouter("request")
	m.router.Register("roomlist", m.handleRoomList)
	m.router.Register("newroom", m.handleNewRoom)

	ideal_port := Config.ReadConfInt("manager_port", 18573)

	var addr, err = net.ResolveTCPAddr("tcp", ":"+strconv.Itoa(ideal_port))
	if err != nil {
		// handle error
		return err
	}

	for i := 0; ; i++ {
		m.ln, err = net.ListenTCP("tcp", addr)
		if err == nil {
			break
		}

		// Retry about 20 times.
		if i >= 19 {
			break
		}
		log.Println("RoomManager is cannot listen on port, sleep and retry...")

		// Each retry sleeps 5 seconds
		time.Sleep(5 * time.Second)
	}

	if err != nil {
		// handle error
		log.Println("RoomManager is cannot listen on port after retry", ideal_port)
		return err
	}

	log.Println("RoomManager is listening on port", ideal_port)

	m.recovery()
	go m.shortenRooms()

	return nil
}

func (m *RoomManager) recovery() error {
	dbDir := Config.ReadConfString("db_dir", "")
	if len(dbDir) <= 0 {
		log.Panicln("db_dir does not present")
	}
	db, err := leveldb.OpenFile(dbDir, nil)
	m.db = db

	iter := db.NewIterator(dbutil.BytesPrefix([]byte("room-")), nil)
	for iter.Next() {
		// Use key/value.
		//key := iter.Key()
		value := iter.Value()
		info := parseRoomRuntimeInfo(value)
		room, err := Room.RecoverRoom(info)
		if err != nil {
			log.Println("room is corrupted", iter.Key())
			continue
		}

		m.rooms.Store(room.Options.Name, room)
		atomic.AddInt32(&m.currentRoomCount, -1)
		go func(room *Room.Room, m *RoomManager) {
			roomName := room.Options.Name
			room.Run()
			m.waitRoomClosed(roomName)
		}(room, m)

	}
	iter.Release()
	err = iter.Error()
	return err
}

func (m *RoomManager) shortenRooms() {
	for {
		select {
		case <-time.After(time.Hour):
			iter := m.db.NewIterator(dbutil.BytesPrefix([]byte("room-")), nil)

			batch := new(leveldb.Batch)

			for iter.Next() {
				value := iter.Value()
				info := parseRoomRuntimeInfo(value)
				if info.Expiration > 1 {
					info.Expiration = info.Expiration - 1

					info_to_insert, err := info.ToJson()
					if err != nil {
						log.Println(err)
						continue
					}
					batch.Put(iter.Key(), info_to_insert)
				} else {
					// delete expired or corrupted room
					batch.Delete(iter.Key())
				}

			}
			iter.Release()
			m.db.Write(batch, nil)
		case _, _ = <-m.goingClose:
			return
		}
	}
}

func (m *RoomManager) waitRoomClosed(roomName string) {
	m.db.Delete([]byte("room-"+roomName), &opt.WriteOptions{})
	m.rooms.Delete(roomName)
	atomic.AddInt32(&m.currentRoomCount, -1)
}

func (m *RoomManager) Close() {
	close(m.goingClose)
	m.db.Close()
	m.ln.Close()
}

func (m *RoomManager) Run() (err error) {
	err = m.init()
	if err != nil {
		return err
	}
	for {
		select {
		case _, _ = <-m.goingClose:
			return err
		default:
			conn, err := m.ln.AcceptTCP()
			if err != nil {
				// handle error
				log.Println(err)
				continue
			}
			go m.processClient(Socket.MakeSocketClient(conn))
		}
	}
}

func (m *RoomManager) processClient(client *Socket.SocketClient) {
    handler := Hub.Handler{
        Name: "roomManagerProcessClient",
        Callback: func(content interface{}) {
            pkg, ok := content.(Socket.Package)
            if !ok {
				return
            }
            if pkg.PackageType == Socket.MANAGER {
				err := m.router.OnMessage(pkg.Unpacked, client)
				if err != nil {
					log.Println(err)
					client.Close()
				}
			}
        },
	}
    client.Sub("package", handler)
}

func ServeManager() *RoomManager {
	return &RoomManager{}
}
