package Room

import (
	"Config"
	"Radio"
	"Router"
	"Socket"
	"log"
	"net"
	"os"
	"path"
	"path/filepath"
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

type RoomUser struct {
	clientId string
	nickName string
}

type Room struct {
	ln          *net.TCPListener
	GoingClose  chan bool
	router      *Router.Router
	radio       *Radio.Radio
	clients     map[*Socket.SocketClient]*RoomUser
	expiration  int
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

func (m *Room) Close() {
	m.locker.Lock()
	defer m.locker.Unlock()
	m.radio.Close()
	m.removeAllClient()
	m.ln.Close()
}

func (m *Room) init() (err error) {
	m.clients = make(map[*Socket.SocketClient]*RoomUser)
	m.GoingClose = make(chan bool)
	m.router = Router.MakeRouter()

	var addr *net.TCPAddr

	if len(m.key) > 0 {
		// recover
		addr, err = net.ResolveTCPAddr("tcp",
			":"+strconv.FormatInt(int64(m.port), 10))
		if err != nil {
			// handle error
			return err
		}
	} else {
		// new-create
		var source = append([]byte(m.Options.Name),
			[]byte(m.Options.Password)...)
		m.key = genSignedKey(source)
		log.Println("key", m.key)
		m.expiration = 48

		addr, err = net.ResolveTCPAddr("tcp", ":0")
		if err != nil {
			// handle error
			return err
		}
		m.archiveSign = genArchiveSign(m.Options.Name)
	}

	data_dir, ok := config["data_dir"].(string)
	if len(data_dir) <= 0 || !ok {
		log.Println("Using default data path", "./data")
		data_dir = "./data/"
	}
	data_path := filepath.Join(data_dir, m.archiveSign+".data")

	if os.MkdirAll(path.Join(data_dir), 0666) != nil {
		log.Println("Cannot make dir", path.Join(data_dir))
		panic(err)
	}

	radio, err := Radio.MakeRadio(data_path)
	m.radio = radio

	m.ln, err = net.ListenTCP("tcp", addr)
	if err != nil {
		// handle error
		// TODO: handle port already in use
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

	m.router.Register("login", m.handleJoin)
	//m.router.Register("heartbeat", m.handleHeartbeat)
	m.router.Register("archivesign", m.handleArchiveSign)
	m.router.Register("archive", m.handleArchive)
	m.router.Register("clearall", m.handleClearAll)
	m.router.Register("kick", m.handleKick)
	//m.router.Register("onlinelist", m.handleOnlineList)

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
	defer m.locker.Unlock()
	return len(m.clients)
}

func (m *Room) OnlineList() (list []OnlineListItem) {
	m.locker.Lock()
	defer m.locker.Unlock()
	for _, user := range m.clients {
		list = append(list, OnlineListItem{
			Name:     user.nickName,
			ClientId: user.clientId,
		})
	}
	return list
}

func (m *Room) Dump() []byte {
	return dumpRoom(m)
}

func (m *Room) hasUser(u *Socket.SocketClient) bool {
	m.locker.Lock()
	defer m.locker.Unlock()
	if user, ok := m.clients[u]; ok && len(user.clientId) > 0 {
		return true
	}
	return false
}

func (m *Room) Run() error {
	log.Println("Room ", m.Options.Name, " is running")
	go func() {
		for {
			select {
			case _, _ = <-m.GoingClose:
				return
			default:
				conn, err := m.ln.AcceptTCP()
				if err != nil {
					// TODO: handle error
					//panic(err)
					continue
				}
				var client = Socket.MakeSocketClient(conn)
				m.locker.Lock()
				m.clients[client] = &RoomUser{}
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
				log.Println("Room is going go close")
				m.removeAllClient()
				return
			case pkg, ok := <-client.PackageChan:
				if !ok {
					m.removeClient(client)
					return
				}
				//go func() {
				switch pkg.PackageType {
				case Socket.COMMAND:
					m.router.OnMessage(pkg.Unpacked, client)
				case Socket.DATA:
					select {
					case m.radio.WriteChan <- Radio.RadioSendPart{
						Data: pkg.Repacked,
					}:
					case <-time.After(time.Second * 5):
						log.Println("WriteChan failed in processClient")
					}
				case Socket.MESSAGE:
					select {
					case m.radio.SendChan <- Radio.RadioSendPart{
						Data: pkg.Repacked,
					}:
					case <-time.After(time.Second * 5):
						log.Println("SendChan failed in processClient")
					}
				}
				//}()
			case _, _ = <-client.GoingClose:
				m.removeClient(client)
				return
			case <-time.After(time.Second * 10):
				log.Println("processClient blocked detected")
				m.kickClient(client)
				return
			}
		}
	}()
}

func (m *Room) removeClient(client *Socket.SocketClient) {
	log.Println("would like to remove client from room")
	m.locker.Lock()
	defer m.locker.Unlock()
	delete(m.clients, client)
	log.Println("client removed from room")
}

func (m *Room) removeAllClient() {
	log.Println("would like to remove client from room")
	m.locker.Lock()
	defer m.locker.Unlock()
	m.clients = make(map[*Socket.SocketClient]*RoomUser)
	log.Println("client removed from room")
}

func (m *Room) kickClient(target *Socket.SocketClient) {
	m.removeClient(target)
	time.AfterFunc(time.Second*10, target.Close)
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

func RecoverRoom(info *RoomRuntimeInfo) (*Room, error) {
	var room = Room{
		port:        info.Port,
		expiration:  info.Expiration,
		archiveSign: info.ArchiveSign,
		key:         info.Key,
		Options:     info.Options,
		locker:      &sync.Mutex{},
	}
	if err := room.init(); err != nil {
		return &Room{}, err
	}

	return &room, nil
}
