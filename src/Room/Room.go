package Room

import (
	"Config"
	"Radio"
	"Router"
	"Socket"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"
)

type RoomOption struct {
	MaxLoad    int
	Width      int64
	Height     int64
	Password   string
	WelcomeMsg string
	Name       string
	EmptyClose bool
}

type Room struct {
	ln          *net.TCPListener
	GoingClose  chan bool
	router      *Router.Router
	radio       *Radio.Radio
	clients     map[*Socket.SocketClient]string
	expiration  int
	salt        string
	key         string
	archiveSign string
	port        uint16
	Options     RoomOption
	locker      *sync.Mutex
}

var config map[interface{}]interface{}

func init() {
	config = Config.GetConfig()
}

func (m *Room) init() error {
	m.clients = make(map[*Socket.SocketClient]string)
	m.GoingClose = make(chan bool)
	m.router = Router.MakeRouter()
	var source = append([]byte(m.Options.Name),
		[]byte(m.Options.Password)...)
	m.key = genSignedKey(source)
	radio, err := Radio.MakeRadio(m.key + ".data")
	m.radio = radio
	m.expiration = 48
	m.router.Register("login", m.handleJoin)
	m.router.Register("heartbeat", m.handleHeartbeat)
	m.router.Register("archivesign", m.handleArchiveSign)
	m.router.Register("archive", m.handleArchive)
	m.router.Register("clearall", m.handleClearAll)

	addr, err := net.ResolveTCPAddr("tcp", ":0")
	if err != nil {
		// handle error
		return err
	}
	m.ln, err = net.ListenTCP("tcp", addr)
	if err != nil {
		// handle error
		return err
	}
	_, port, err := net.SplitHostPort(m.ln.Addr().String())
	if err != nil {
		return err
	}
	uport, err := strconv.ParseUint(port, 10, 16)
	m.port = uint16(uport)
	if err != nil {
		return err
	}
	fmt.Println("port", m.port)
	return nil
}

func (m *Room) Port() uint16 {
	return m.port
}

func (m *Room) Key() string {
	return m.key
}

func (m *Room) CurrentLoad() int {
	m.locker.Lock()
	var length = len(m.clients)
	m.locker.Unlock()
	return length
}

func (m *Room) Run() error {
	fmt.Println("Room ", m.Options.Name, " is running")
	go func() {
		for {
			select {
			case _, _ = <-m.GoingClose:
				return
			default:
				conn, err := m.ln.AcceptTCP()
				if err != nil {
					// handle error
					panic(err)
					continue
				}
				var client = Socket.MakeSocketClient(conn)
				m.locker.Lock()
				m.clients[client] = m.genClientId()
				m.locker.Unlock()
				m.processClient(client)
			}
		}
	}()
	return nil
}

func (m *Room) processClient(client *Socket.SocketClient) {
	go func() {
		for {
			select {
			case _, _ = <-m.GoingClose:
				fmt.Println("Room is going go close")
				return
			case pkg, ok := <-client.PackageChan:
				if !ok {
					return
				}
				go func() {
					switch pkg.PackageType {
					case Socket.COMMAND:
						m.router.OnMessage(pkg.Unpacked, client)
					case Socket.DATA:
						select {
						case m.radio.WriteChan <- Radio.RadioSendPart{
							Data: pkg.Repacked,
						}:
						case <-time.After(time.Second * 5):
							fmt.Println("WriteChan failed in processClient")
						}
					case Socket.MESSAGE:
						select {
						case m.radio.SendChan <- Radio.RadioSendPart{
							Data: pkg.Repacked,
						}:
						case <-time.After(time.Second * 5):
							fmt.Println("SendChan failed in processClient")
						}
					}
				}()
			case _, _ = <-client.GoingClose:
				m.locker.Lock()
				delete(m.clients, client)
				m.locker.Unlock()
				fmt.Println("client removed from room")
				return
			}
		}
	}()
}

func ServeRoom(opt RoomOption) (*Room, error) {
	var room = Room{
		Options: opt,
		locker:  &sync.Mutex{},
	}
	if err := room.init(); err != nil {
		return &Room{}, err
	}

	return &room, nil
}
