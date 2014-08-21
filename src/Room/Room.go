package Room

import (
	"Config"
	"Radio"
	"Router"
	"Socket"
	"bytes"
	xxhash "github.com/OneOfOne/xxhash/native"
	"io"
	"net"
	"strconv"
)

type RoomOption struct {
	MaxLoad    int
	Width      int
	Height     int
	Password   string
	WelcomeMsg string
	Name       string
	EmptyClose bool
}

type Room struct {
	ln          *net.TCPListener
	GoingClose  chan bool
	router      Router.Router
	radio       Radio.Radio
	clients     []Socket.SocketClient
	expiration  int
	salt        string
	key         string
	archiveSign string
	port        uint16
	Options     RoomOption
}

var config map[string]interface{}

func init() {
	config = Config.GetConfig()
}

func genSignedKey(source []byte) string {
	h := xxhash.New64()
	r := bytes.NewReader(append(source, config["globalSaltHash"].([]byte)...))
	io.Copy(h, r)
	hash := h.Sum64()
	return strconv.FormatUint(hash, 32)
}

func (m *Room) init() error {
	m.clients = make([]Socket.SocketClient, 0, 10)
	m.GoingClose = make(chan bool, 1)
	m.router = Router.MakeRouter()
	var source = append([]byte(m.Options.Name),
		[]byte(m.Options.Password)...)
	m.key = genSignedKey(source)
	radio, err := Radio.MakeRadio(m.key + ".data")
	m.radio = radio
	m.expiration = 48
	//m.router.Register("roomlist", m.handleRoomList)

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
			case <-m.GoingClose:
				m.GoingClose <- true // feed to other goros
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
	m.radio.AddClient(client, 0, m.radio.FileSize())
	go func() {
		for {
			select {
			case <-m.GoingClose:
				m.GoingClose <- true
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

func ServeRoom(opt RoomOption) (Room, error) {
	var room = Room{
		Options: opt,
	}
	if err := room.init(); err != nil {
		return Room{}, err
	}

	return room, nil
}
