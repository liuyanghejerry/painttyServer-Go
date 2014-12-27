package Room

import (
	//"Radio"
	"Socket"
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	xxhash "github.com/OneOfOne/xxhash/native"
	"github.com/dustin/randbo"
	"io"
	//"log"
	"strconv"
	"strings"
	"time"
)

type RoomRuntimeInfo struct {
	Key         string     `json: "key"`
	ArchiveSign string     `json: "archiveSign"`
	Port        uint16     `json: "port"`
	Expiration  int        `json: "expiration"`
	Options     RoomOption `json: "options"`
}

func (r *RoomRuntimeInfo) ToJson() ([]byte, error) {
	return json.Marshal(*r)
}

func dumpRoom(room *Room) []byte {

	info := RoomRuntimeInfo{
		Key:         room.key,
		ArchiveSign: room.archiveSign,
		Expiration:  room.expiration,
		Port:        room.port,
		Options:     room.Options,
	}

	raw, err := info.ToJson()
	if err != nil {
		panic(err)
	}

	return raw
}

func (m *Room) findClientById(clientId string) *Socket.SocketClient {
	m.locker.Lock()
	defer m.locker.Unlock()
	for client, user := range m.clients {
		if user.clientId == clientId {
			return client
		}
	}
	return nil
}

func (m *Room) broadcastCommand(resp interface{}) {
	data, err := json.Marshal(resp)
	if err != nil {
		panic(err)
	}

	m.locker.Lock()
	defer m.locker.Unlock()

	for cli, usr := range m.clients {
		if len(usr.clientId) > 0 {
			_, err = cli.SendCommandPack(data)
			if err != nil {
				cli.Close()
			}
		}
	}
}

func (m *Room) sendAnnouncement(client *Socket.SocketClient) {
	msg := config["announcement"].(string)
	if len(msg) <= 0 {
		return
	}
	resp := NotifyAction{
		Action:  "notify",
		Content: msg + "\n",
	}
	directSendCommand(resp, client)
}

func (m *Room) sendWelcomeMsg(client *Socket.SocketClient) {
	msg := m.Options.WelcomeMsg
	if len(msg) <= 0 {
		return
	}
	resp := WelcomeMsgType{
		Content: msg + "\n",
	}
	directSendMessage(resp, client)
}

func (m *Room) sendExpirationMsg(client *Socket.SocketClient) {
	leftTime := time.Hour*time.Duration(m.expiration) - time.Since(m.lastCheck)
	msg := fmt.Sprintf("这间房间还剩下约%d小时", int64(leftTime/time.Hour))
	resp := WelcomeMsgType{
		Content: msg + "\n",
	}
	directSendMessage(resp, client)
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

func genArchiveSign(name string) string {
	h := xxhash.New64()
	var buf = make([]byte, 16)
	randbo.New().Read(buf)
	r := strings.NewReader(name + hex.EncodeToString(buf))
	io.Copy(h, r)
	hash := h.Sum64()
	return strconv.FormatUint(hash, 32)
}
