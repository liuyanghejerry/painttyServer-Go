package RoomManager

import (
	//"fmt"
	//"log"
	"net"
	//"time"
	"Router"
	"Socket"
	//"encoding/json"
	"sync"
)

type RoomManager struct {
	clients    []Socket.SocketClient
	ln         *net.TCPListener
	goingClose chan bool
	router     Router.Router
}

func (m *RoomManager) init() error {
	m.clients = make([]Socket.SocketClient, 0, 100)
	m.goingClose = make(chan bool, 1)
	m.router = Router.MakeRouter()
	m.router.Register("roomlist", m.handleRoomList)

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
			case <-m.goingClose:
				m.goingClose <- true // feed to other goros
				return
			default:
				conn, err := m.ln.AcceptTCP()
				if err != nil {
					// handle error
					continue
				}
				var client = Socket.MakeSocketClient(conn)
				m.clients = append(m.clients, client)
				go m.processClient(&client)
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
			case <-m.goingClose:
				m.goingClose <- true
				return
			case pkg := <-client.PackageChan:
				if pkg.PackageType == Socket.MANAGER {
					m.router.OnMessage(pkg.Unpacked, client)
				}
				break
			case <-client.GoingClose:
				client.GoingClose <- true
				return
			}
		}
	}()
}

func ServeManager() RoomManager {
	return RoomManager{}
}
