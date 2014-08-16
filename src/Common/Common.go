package Common

import "bytes"
import "compress/flate"
import "io"

func QCompress(data []byte) ([]byte, error) {
	var length = len(data)
	var tmp bytes.Buffer

	tmp.WriteByte(byte(length >> 24))
	tmp.WriteByte(byte((length & 0x00FF0000) >> 16))
	tmp.WriteByte(byte((length & 0x0000FF00) >> 8))
	tmp.WriteByte(byte(length & 0xFF))

	var in bytes.Buffer

	w, err := flate.NewWriter(&in, -1)
	if err != nil {
		return []byte{}, err
	}
	w.Write(data)
	w.Close()

	tmp.Write(in.Bytes())

	return tmp.Bytes(), nil
}

func QUncompress(data []byte) ([]byte, error) {
	var resized = bytes.NewBuffer(data[4:])
	var tmp bytes.Buffer
	r := flate.NewReader(resized)
	io.Copy(&tmp, r)
	r.Close()

	return tmp.Bytes(), nil
}
