package Room

import "github.com/dustin/randbo"
import "encoding/hex"

func genClientId() string {
	var buf = make([]byte, 64)
	randbo.New().Read(buf)
	return hex.EncodeToString(buf)
}
