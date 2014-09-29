package Room

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	xxhash "github.com/OneOfOne/xxhash/native"
	"github.com/dustin/randbo"
	"io"
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

func dumpRoom(room *Room) []byte {

	info := RoomRuntimeInfo{
		Key:         room.key,
		ArchiveSign: room.archiveSign,
		Expiration:  room.expiration,
		Port:        room.port,
		Options:     room.Options,
	}

	raw, err := json.Marshal(info)
	if err != nil {
		panic(err)
	}

	return raw
}

func genClientId() string {
	var buf = make([]byte, 64)
	randbo.New().Read(buf)
	return hex.EncodeToString(buf)
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
