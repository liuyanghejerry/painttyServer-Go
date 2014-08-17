package Room

import (
	"Router"
	"Socket"
	"net"
)

type Room struct {
	ln          *net.TCPListener
	goingClose  chan bool
	router      Router.Router
	clients     []Socket.SocketClient
	maxload     int
	password    string
	welcomemsg  string
	emptyclose  bool
	expiration  int
	salt        string
	key         string
	archiveSign string
	port        int
}

func (m *Room) init() error {
	m.clients = make([]Socket.SocketClient, 0, 10)
	m.goingClose = make(chan bool, 1)
	m.router = Router.MakeRouter()
	m.maxload = 8
	m.emptyclose = true
	m.expiration = 48
	//m.router.Register("roomlist", m.handleRoomList)

	var addr, err = net.ResolveTCPAddr("tcp", ":0")
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

func (m *Room) Run() error {
	err := m.init()
	if err != nil {
		return err
	}
	go func() {
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
	return nil
}

func (m *Room) processClient(client *Socket.SocketClient) {
	go func() {
		for {
			select {
			case <-m.goingClose:
				m.goingClose <- true
				return
			case pkg := <-client.PackageChan:
				switch pkg.PackageType {
				case Socket.COMMAND:
					m.router.OnMessage(pkg.Unpacked, client)
					break
				case Socket.DATA:
					// TODO
					//m.router.OnMessage(pkg.Unpacked, client)
					break
				case Socket.MESSAGE:
					// TODO
					//m.router.OnMessage(pkg.Unpacked, client)
					break
				}
				break
			case <-client.GoingClose:
				client.GoingClose <- true
				return
			}
		}
	}()
}

func ServeRoom() Room {
	return Room{}
}
