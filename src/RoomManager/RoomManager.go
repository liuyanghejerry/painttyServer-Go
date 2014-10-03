package RoomManager

import (
	"Room"
	"Router"
	"Socket"
	"fmt"
	"github.com/syndtr/goleveldb/leveldb"
	dbutil "github.com/syndtr/goleveldb/leveldb/util"
	"net"
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

	var addr, err = net.ResolveTCPAddr("tcp", ":8080")
	if err != nil {
		// handle error
		return err
	}
	m.ln, err = net.ListenTCP("tcp", addr)
	if err != nil {
		// handle error
		return err
	}

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
		fmt.Println("Room recovered", string(value))
		if err != nil {
			panic(err)
		}

		m.roomsLocker.Lock()
		m.rooms[room.Options.Name] = room
		m.roomsLocker.Unlock()
		room.Run()
		go func() {
			_, _ = <-room.GoingClose
			m.roomsLocker.Lock()
			delete(m.rooms, room.Options.Name)
			m.roomsLocker.Unlock()
			return
		}()
	}
	iter.Release()
	err = iter.Error()
	return err
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
