package Common

import "bytes"
import "compress/zlib"
import "io"

func QCompress(data []byte) ([]byte, error) {
	var length = len(data)
	var tmp bytes.Buffer

	tmp.WriteByte(byte(length >> 24))
	tmp.WriteByte(byte((length & 0x00FF0000) >> 16))
	tmp.WriteByte(byte((length & 0x0000FF00) >> 8))
	tmp.WriteByte(byte(length & 0xFF))

	var in bytes.Buffer

	w := zlib.NewWriter(&in)
	w.Write(data)
	w.Close()

	tmp.Write(in.Bytes())

	return tmp.Bytes(), nil
}

func QUncompress(data []byte) (result []byte, err error) {
	defer func() {
		// recover from panic if one occured. Set err to nil otherwise.
		if e := recover(); e != nil {
			err = e.(error)
			result = []byte{}
		}
	}()
	var resized = bytes.NewBuffer(data[4:])
	var tmp bytes.Buffer
	r, err := zlib.NewReader(resized)
	if err != nil {
		return []byte{}, err
	}
	io.Copy(&tmp, r)
	r.Close()

	return tmp.Bytes(), nil
}
