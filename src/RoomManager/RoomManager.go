package RoomManager

import (
	"Config"
	"Room"
	"Router"
	"Socket"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	dbutil "github.com/syndtr/goleveldb/leveldb/util"
	cDebug "github.com/tj/go-debug"
	"log"
	"net"
	"strconv"
	"sync"
	//"time"
)

var debugOut = cDebug.Debug("RoomManager")

type RoomManager struct {
	ln          *net.TCPListener
	goingClose  chan bool
	router      *Router.Router
	rooms       map[string]*Room.Room
	roomsLocker sync.Mutex
	db          *leveldb.DB
}

func (m *RoomManager) init() error {
	m.goingClose = make(chan bool)
	m.rooms = make(map[string]*Room.Room)
	m.router = Router.MakeRouter("request")
	m.router.Register("roomlist", m.handleRoomList)
	m.router.Register("newroom", m.handleNewRoom)

	ideal_port, ok := Config.GetConfig()["manager_port"].(int)
	if ideal_port <= 0 || !ok {
		log.Println("Manager port is not configured, using default ", ideal_port)
		ideal_port = 18573
	}

	var addr, err = net.ResolveTCPAddr("tcp", ":"+strconv.Itoa(ideal_port))
	if err != nil {
		// handle error
		return err
	}
	m.ln, err = net.ListenTCP("tcp", addr)
	if err != nil {
		// handle error
		return err
	}

	log.Println("RoomManager is listening on port", ideal_port)

	m.recovery()

	return nil
}

func (m *RoomManager) recovery() error {
	db, err := leveldb.OpenFile("./db", nil)
	m.db = db

	iter := db.NewIterator(dbutil.BytesPrefix([]byte("room-")), nil)
	for iter.Next() {
		// Use key/value.
		//key := iter.Key()
		value := iter.Value()
		info := parseRoomRuntimeInfo(value)
		room, err := Room.RecoverRoom(info)
		debugOut("Room recovered", string(value))
		if err != nil {
			panic(err)
		}

		m.roomsLocker.Lock()
		m.rooms[room.Options.Name] = room
		m.roomsLocker.Unlock()
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

func (m *RoomManager) waitRoomClosed(roomName string) {
	log.Println(roomName, "is closed.")
	m.roomsLocker.Lock()
	defer m.roomsLocker.Unlock()
	delete(m.rooms, roomName)
	m.db.Delete([]byte("room-"+roomName), &opt.WriteOptions{false})
}

func (m *RoomManager) Close() {
	m.roomsLocker.Lock()
	defer m.roomsLocker.Unlock()
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
				continue
			}
			m.processClient(Socket.MakeSocketClient(conn))
		}
	}
	return err
}

func (m *RoomManager) processClient(client *Socket.SocketClient) {
	go func() {
		for {
			select {
			case _, _ = <-m.goingClose:
				return
			case pkg, ok := <-client.PackageChan:
				if !ok {
					return
				}
				if pkg.PackageType == Socket.MANAGER {
					err := m.router.OnMessage(pkg.Unpacked, client)
					if err != nil {
						client.Close()
					}
				}
			case _, _ = <-client.GoingClose:
				return
			}
		}
	}()
}

func ServeManager() *RoomManager {
	return &RoomManager{}
}
