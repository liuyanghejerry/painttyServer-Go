package Socket

import "testing"
import "bytes"
import "Common"

var source = []byte{1, 2}

func TestProtocolPack(t *testing.T) {
	var result = protocolPack(source)
	if bytes.Compare(result, []byte{0, 0, 0, 2, 1, 2}) != 0 {
		t.Error("protocolPack error", result)
	}
}

func TestBufferToPack(t *testing.T) {
	var header = PackHeader{
		true,
		DATA,
	}
	result, err := bufferToPack(source, header)
	if err != nil {
		t.Error("bufferToPack error", err)
	}

	converted, err := Common.QCompress(source)
	if err != nil {
		t.Error("bufferToPack error", err)
	}

	var expected bytes.Buffer
	expected.WriteByte(0x05)
	expected.Write(converted)

	if bytes.Compare(result, expected.Bytes()) != 0 {
		t.Error("bufferToPack error", result, expected.Bytes())
	}
}
