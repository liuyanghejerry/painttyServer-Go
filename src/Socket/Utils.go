package Socket

import "Common"
import "bytes"

const ( // iota is reset to 0
	MANAGER = iota    // 0
	COMMAND = iota    // 1
	DATA    = iota    // 2
	MESSAGE = iota    // 3
	MASK    = MESSAGE // 3
)

type PackHeader struct {
	Compress bool
	PackType int
}

func protocolPack(data []byte) []byte {
	var length = len(data)
	var tmp bytes.Buffer
	c1 := byte(length & 0xFF)
	length >>= 8
	c2 := byte(length & 0xFF)
	length >>= 8
	c3 := byte(length & 0xFF)
	length >>= 8
	c4 := byte(length & 0xFF)
	tmp.WriteByte(c4)
	tmp.WriteByte(c3)
	tmp.WriteByte(c2)
	tmp.WriteByte(c1)
	tmp.Write(data)
	return tmp.Bytes()
}

func bufferToPack(data []byte, header PackHeader) ([]byte, error) {
	var converted []byte
	// compress if it requires
	if header.Compress {
		var err error
		converted, err = Common.QCompress(data)
		if err != nil {
			return []byte{}, err
		}
	} else {
		converted = data
	}

	// add header
	var tmpData bytes.Buffer
	var compress_bit byte
	if header.Compress {
		compress_bit = byte(0x1)
	} else {
		compress_bit = byte(0x0)
	}
	var pack_type_bits = byte((header.PackType & MASK) << 0x1)
	var header_bits = compress_bit | pack_type_bits
	tmpData.WriteByte(header_bits)
	tmpData.Write(converted)

	return tmpData.Bytes(), nil
}
