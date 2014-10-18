package RoomManager

import (
	"Config"
	"Room"
	"Router"
	"Socket"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	dbutil "github.com/syndtr/goleveldb/leveldb/util"
	"log"
	"net"
	"strconv"
	"sync"
	//"time"
)

type RoomManager struct {
	clients     []*Socket.SocketClient
	ln          *net.TCPListener
	goingClose  chan bool
	router      *Router.Router
	rooms       map[string]*Room.Room
	roomsLocker sync.Mutex
	db          *leveldb.DB
}

func (m *RoomManager) init() error {
	m.clients = make([]*Socket.SocketClient, 0, 100)
	m.goingClose = make(chan bool)
	m.rooms = make(map[string]*Room.Room)
	m.router = Router.MakeRouter()
	m.router.Register("roomlist", m.handleRoomList)
	m.router.Register("newroom", m.handleNewRoom)

	config := Config.GetConfig()

	ideal_port, ok := config["manager_port"].(int)
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
		log.Println("Room recovered", string(value))
		if err != nil {
			panic(err)
		}

		m.roomsLocker.Lock()
		m.rooms[room.Options.Name] = room
		m.roomsLocker.Unlock()
		go func(room *Room.Room) {
			roomName := room.Options.Name
			room.Run()
			m.waitRoomClosed(roomName)
		}(room)

	}
	iter.Release()
	err = iter.Error()
	return err
}

func (m *RoomManager) waitRoomClosed(roomName string) {
	log.Println(roomName, "is closed.")
	m.roomsLocker.Lock()
	delete(m.rooms, roomName)
	m.roomsLocker.Unlock()
	m.db.Delete([]byte("room-"+roomName), &opt.WriteOptions{false})
}

func (m *RoomManager) Close() {
	m.roomsLocker.Lock()
	defer m.roomsLocker.Unlock()
	close(m.goingClose)
	m.db.Close()
	for _, room := range m.clients {
		room.Close()
	}
	m.ln.Close()
}

func (m *RoomManager) Run() (err error) {
	err = m.init()
	if err != nil {
		return err
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case _, _ = <-m.goingClose:
				return
			default:
				conn, err := m.ln.AcceptTCP()
				if err != nil {
					// handle error
					continue
				}
				var client = Socket.MakeSocketClient(conn)
				m.clients = append(m.clients, client)
				m.processClient(client)
			}

		}
	}()
	wg.Wait()
	return err
}

func (m *RoomManager) processClient(client *Socket.SocketClient) {
	go func() {
		for {
			select {
			case _, _ = <-m.goingClose:
				return
			case pkg, ok := <-client.PackageChan:
				if ok {
					if pkg.PackageType == Socket.MANAGER {
						m.router.OnMessage(pkg.Unpacked, client)
					}
				} else {
					return
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
