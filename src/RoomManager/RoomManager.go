package RoomManager

import (
	//"fmt"
	//"log"
	"Room"
	"Router"
	"Socket"
	"net"
	"sync"
)

type RoomManager struct {
	clients     []*Socket.SocketClient
	ln          *net.TCPListener
	goingClose  chan bool
	router      *Router.Router
	rooms       map[string]*Room.Room
	roomsLocker sync.Mutex
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
	return nil
}

func (m *RoomManager) Run() error {
	err := m.init()
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
	return nil
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
