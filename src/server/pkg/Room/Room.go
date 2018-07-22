package Room

import (
	"errors"
	cDebug "github.com/visionmedia/go-debug"
	"log"
	"net"
	"os"
	"path"
	"path/filepath"
	"server/pkg/Config"
	"server/pkg/Radio"
	"server/pkg/Router"
	"server/pkg/Socket"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

var debugOut = cDebug.Debug("Room")

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
	ln                  *net.TCPListener
	GoingClose          chan bool
	router              *Router.Router
	radio               *Radio.Radio
	clients             sync.Map
	currentClientsCount uint32
	expiration          int
	key                 string
	archiveSign         string
	port                uint16
	Options             RoomOption
	lastCheck           time.Time
}

func (m *Room) Close() {
	debugOut("would like to close Room")
	close(m.GoingClose)
	debugOut("room channel closed")
	m.radio.Close()
	debugOut("room radio closed")
	m.radio.Remove()
	debugOut("room radio removed")
	m.ln.Close()
	debugOut("room connection closed")
}

func (m *Room) init() (err error) {
	m.GoingClose = make(chan bool)
	m.router = Router.MakeRouter("request")

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
		debugOut("key", m.key)

		addr, err = net.ResolveTCPAddr("tcp", ":0")
		if err != nil {
			// handle error
			return err
		}
		m.archiveSign = genArchiveSign(m.Options.Name)
	}

	data_dir, ok := Config.GetConfig()["data_dir"].(string)
	if len(data_dir) <= 0 || !ok {
		log.Println("Using default data path", "./data")
		data_dir = "./data/"
	}
	data_path := filepath.Join(data_dir, m.archiveSign+".data")

	if os.MkdirAll(path.Join(data_dir), 0666) != nil {
		debugOut("Cannot make dir", path.Join(data_dir))
		panic(err)
	}

	radio, err := Radio.MakeRadio(data_path, m.archiveSign)
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
	m.router.Register("heartbeat", m.handleHeartbeat)
	m.router.Register("archivesign", m.handleArchiveSign)
	m.router.Register("archive", m.handleArchive)
	m.router.Register("clearall", m.handleClearAll)
	m.router.Register("kick", m.handleKick)
	m.router.Register("onlinelist", m.handleOnlineList)
	m.router.Register("close", m.handleClose)
	m.router.Register("checkout", m.handleCheckout)

	return nil
}

func (m *Room) Port() uint16 {
	return m.port
}

func (m *Room) Key() string {
	return m.key
}

func (m *Room) Password() string {
	return m.Options.Password
}

func (m *Room) CurrentLoad() int {
	return int(atomic.LoadUint32(&m.currentClientsCount))
}

func (m *Room) OnlineList() (list []OnlineListItem) {
	m.clients.Range(func(key, value interface{}) bool {
		user, ok := value.(*RoomUser)
		if !ok {
			return true
		}
		list = append(list, OnlineListItem{
			Name:     user.nickName,
			ClientId: user.clientId,
		})
		return true
	})
	return list
}

func (m *Room) Dump() []byte {
	return dumpRoom(m)
}

func (m *Room) hasUser(u *Socket.SocketClient) bool {
	value, ok := m.clients.Load(u)
	if !ok {
		return false
	}
	user, ok := value.(*RoomUser)

	if !ok {
		return false
	}

	if len(user.clientId) > 0 {
		return true
	}

	return false
}

func (m *Room) processEmptyClose() {
	clientLen := atomic.LoadUint32(&m.currentClientsCount)
	if clientLen == 0 && m.Options.EmptyClose {
		m.Close()
	}
}

func (m *Room) processExpire() {
	for {
		select {
		case _, _ = <-m.GoingClose:
			return
		case <-time.After(time.Hour):
			if time.Since(m.lastCheck) > time.Hour*time.Duration(m.expiration) {
				clientLen := atomic.LoadUint32(&m.currentClientsCount)
				if clientLen == 0 {
					m.Close()
				} else {
					m.Options.EmptyClose = true
				}
			}
		}
	}
}

func (m *Room) Run() error {
	debugOut("Room ", m.Options.Name, " is running")
	go m.processExpire()
	for {
		select {
		case _, _ = <-m.GoingClose:
			return nil
		default:
			conn, err := m.ln.AcceptTCP()
			if err != nil {
				// TODO: handle error
				//panic(err)
				continue
			}
			var client = Socket.MakeSocketClient(conn)
			m.clients.Store(client, &RoomUser{})
			atomic.AddUint32(&m.currentClientsCount, 1)
			m.processClient(client)
		}
	}
}

func (m *Room) processClient(client *Socket.SocketClient) {
	go func() {
		for {
			if client.IsClosed() {
				m.removeClient(client)
				m.processEmptyClose()
				return
			}
			select {
			case _, _ = <-m.GoingClose:
				debugOut("Room is going go close")
				m.removeAllClient()
				return
			case pkg, ok := <-client.GetPackageChan():
				if !ok {
					m.removeClient(client)
					m.processEmptyClose()
					return
				}
				switch pkg.PackageType {
				case Socket.COMMAND:
					err := m.router.OnMessage(pkg.Unpacked, client)
					if err != nil {
						m.kickClient(client)
					}
				case Socket.DATA:
					if !m.hasUser(client) {
						return
					}
					select {
					case m.radio.WriteChan <- Radio.RadioSendPart{
						Data: pkg.Repacked,
					}:
					case <-time.After(time.Second * 5):
						debugOut("WriteChan failed in processClient")
					}
				case Socket.MESSAGE:
					if !m.hasUser(client) {
						return
					}
					select {
					case m.radio.SendChan <- Radio.RadioSendPart{
						Data: pkg.Repacked,
					}:
					case <-time.After(time.Second * 5):
						debugOut("SendChan failed in processClient")
					}
				}
			case <-time.After(time.Second * 10):
				debugOut("processClient blocked detected")
				m.kickClient(client)
				return
			}
		}
	}()
}

func (m *Room) removeClient(client *Socket.SocketClient) {
	debugOut("would like to remove client from room")
	m.clients.Delete(client)
	atomic.AddUint32(&m.currentClientsCount, ^uint32(0))
	m.radio.RemoveClient(client)
	debugOut("client removed from room")
}

func (m *Room) removeAllClient() {
	debugOut("would like to remove all clients from room")
	m.removeAllClient_internal()
}

func (m *Room) removeAllClient_internal() {
	debugOut("would like to remove all clients from room")
	m.clients = sync.Map{}
	m.radio.RemoveAllClients()
	debugOut("all clients removed from room")
}

func (m *Room) kickClient(target *Socket.SocketClient) {
	m.removeClient(target)
	time.AfterFunc(time.Second*10, target.Close)
}

func ServeRoom(opt RoomOption) (*Room, error) {
	var room = Room{
		Options:    opt,
		expiration: Config.GetConfig()["expiration"].(int),
		lastCheck:  time.Now(),
	}
	if err := room.init(); err != nil {
		return &Room{}, err
	}

	return &room, nil
}

func RecoverRoom(info *RoomRuntimeInfo) (r *Room, err error) {
	var room = Room{
		port:        info.Port,
		expiration:  info.Expiration,
		archiveSign: info.ArchiveSign,
		key:         info.Key,
		Options:     info.Options,
		lastCheck:   time.Now(),
	}
	if err := room.init(); err != nil {
		return &Room{}, err
	}

	defer func() {
		if err := recover(); err != nil {
			log.Println("room recover encountered panic", info.Options.Name, err)
			r = nil
			err = errors.New("Room recover failure.")
		}
	}()

	return &room, nil
}
