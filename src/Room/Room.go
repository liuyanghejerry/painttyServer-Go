package Room

import (
	"Config"
	"Radio"
	"Router"
	"Socket"
	"bytes"
	"fmt"
	xxhash "github.com/OneOfOne/xxhash/native"
	"io"
	"net"
	"strconv"
	"sync"
	"time"
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
	clients     map[*Socket.SocketClient]string
	expiration  int
	salt        string
	key         string
	archiveSign string
	port        uint16
	Options     RoomOption
	locker      sync.Mutex
}

var config map[interface{}]interface{}

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

func (m *Room) genClientId() string {
	h := xxhash.New64()
	timeData, err := time.Now().MarshalBinary()
	if err != nil {
		timeData = []byte("asdasdasdfuweyfiaiuehmoixzwe")
	}
	var source = append(timeData, []byte(m.Options.Name)...)
	source = append(source, config["globalSaltHash"].([]byte)...)
	r := bytes.NewReader(source)
	io.Copy(h, r)
	hash := h.Sum64()
	return strconv.FormatUint(hash, 32)
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
			case <-m.GoingClose:
				m.GoingClose <- true // feed to other goros
				return
			default:
				conn, err := m.ln.AcceptTCP()
				fmt.Println("Room got one client")
				if err != nil {
					// handle error
					panic(err)
					continue
				}
				var client = Socket.MakeSocketClient(conn)
				m.locker.Lock()
				m.clients[&client] = m.genClientId()
				m.locker.Unlock()
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
					m.radio.WriteChan <- Radio.RadioSendPart{
						Data: pkg.Repacked,
					}
					break
				case Socket.MESSAGE:
					m.radio.SendChan <- Radio.RadioSendPart{
						Data: pkg.Repacked,
					}
					break
				}
				break
			case <-client.GoingClose:
				client.GoingClose <- true
				m.locker.Lock()
				delete(m.clients, client)
				m.locker.Unlock()
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
