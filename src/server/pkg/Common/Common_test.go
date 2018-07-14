package Common

import "testing"
import "bytes"

var source = []byte{1, 2, 3, 4, 5, 6, 7, 7, 8, 9, 10}
var source2 = []byte(`{"request":"roomlist"}`)

func TestQCompress(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	var result, err = QCompress(source)
	if err != nil {
		t.Error(err)
	}
	if len(result) == 0 {
		t.Error("Compressed content is empty!")
	}
	result, err = QUncompress(result)
	if err != nil {
		t.Error(err)
	}

	if bytes.Compare(result, source) != 0 {
		t.Error("Compressed content not equal to Uncompressed one", result)
	}
}

func TestQCompress2(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	var result, err = QCompress(source2)
	t.Log(result)
	if err != nil {
		t.Error(err)
	}
	if len(result) == 0 {
		t.Error("Compressed content is empty!")
	}
	result, err = QUncompress(result)
	if err != nil {
		t.Error(err)
	}

	if bytes.Compare(result, source2) != 0 {
		t.Error("Compressed content not equal to Uncompressed one", result)
	}
}

func TestQCompress3(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	var result, err = QUncompress([]byte{1, 2, 3})
	t.Log(result, err)
	if err == nil {
		t.Error("cannot recover or detect invalid input")
	}
}
