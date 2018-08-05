package Room

import (
	//"Radio"
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	xxhash "github.com/cespare/xxhash"
	"github.com/dustin/randbo"
	"io"
	"server/pkg/Config"
	"server/pkg/Socket"
	"log"
	"strconv"
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
	var result *Socket.SocketClient
	m.clients.Range(func(key, value interface{}) bool {
		client, ok := key.(*Socket.SocketClient)
		if !ok {
			return true
		}
		user, ok := value.(*RoomUser)
		if !ok {
			return true
		}
		if user.clientId == clientId {
			result = client
			return false
		}
		return true
	})
	return result
}

func (m *Room) broadcastCommand(resp interface{}) {
	data, err := json.Marshal(resp)
	if err != nil {
		panic(err)
	}

	m.clients.Range(func(key, value interface{}) bool {
		client, ok := key.(*Socket.SocketClient)
		if !ok {
			return true
		}
		user, ok := value.(*RoomUser)
		if !ok {
			return true
		}
		if len(user.clientId) > 0 {
			_, err = client.SendCommandPack(data)
			if err != nil {
				client.Close()
			}
		}
		return true
	})
}

func (m *Room) sendAnnouncement(client *Socket.SocketClient) {
	msg := Config.ReadConfString("announcement", "")
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
    lastCheck, ok := m.lastCheck.Load().(time.Time)
	if !ok {
        log.Panicln("lastCheck type assert failed.")
    }
    leftTime := time.Hour*time.Duration(m.expiration) - time.Since(lastCheck)
	msg := fmt.Sprintf("这间房间还剩下约%d小时", int64(leftTime/time.Hour))
	resp := WelcomeMsgType{
		Content: msg + "\n",
	}
	directSendMessage(resp, client)
}

func genSignedKey(source []byte) string {
	h := xxhash.New()
	r := bytes.NewReader(append(source, Config.ReadConfBytes("globalSaltHash")...))
	io.Copy(h, r)
	hash := h.Sum64()
	return strconv.FormatUint(hash, 32)
}

func (m *Room) genClientId() string {
	h := xxhash.New()
	timeData, err := time.Now().MarshalBinary()
	if err != nil {
		timeData = []byte("asdasdasdfuweyfiaiuehmoixzwe")
	}
	var source = append(timeData, []byte(m.Options.Name)...)
	source = append(source, Config.ReadConfBytes("globalSaltHash")...)
	r := bytes.NewReader(source)
	io.Copy(h, r)
	hash := h.Sum64()
	return strconv.FormatUint(hash, 32)
}

func genArchiveSign(name string) string {
	var buf = make([]byte, 16)
	randbo.New().Read(buf)
	hash := xxhash.Sum64String(name + hex.EncodeToString(buf))
	return strconv.FormatUint(hash, 32)
}
