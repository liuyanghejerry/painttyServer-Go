package Room

import (
	"bytes"
	"encoding/hex"
	xxhash "github.com/OneOfOne/xxhash/native"
	"github.com/dustin/randbo"
	"io"
	"strconv"
	"time"
)

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
