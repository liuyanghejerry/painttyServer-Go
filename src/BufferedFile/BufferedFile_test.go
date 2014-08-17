package BufferedFile

import "testing"

//import "bytes"
import "os"
import "time"
import "fmt"

var (
	toWrite  = []byte{1, 2, 3, 4, 5}
	fileName = "./test.file"
)

func init() {
	for i := 0; i < 10; i++ {
		toWrite = append(toWrite, toWrite...)
	}
}

func TestWrite(t *testing.T) {
	var opt = BufferedFileOption{
		fileName,
		time.Second * 1,
		512,
	}

	var file, err = MakeBufferedFile(opt)
	defer func() {
		file.Close()
		os.Remove(fileName)
	}()

	if err != nil {
		t.Error(err)
	}

	if len(file.buffer) != opt.bufferSize {
		t.Log("buffer size is incorrect")
		t.Error(len(file.buffer))
	}

	num, err := file.Write(toWrite)
	fmt.Println(num)
	if num != len(toWrite) || err != nil {
		t.Log("write size error")
		t.Error(num, ", expect:", len(toWrite))
	}

	err = file.Sync()
	if err != nil {
		t.Log("Sync error")
		t.Error(err)
	}

	info, _ := os.Stat(fileName)
	if info.Size() != int64(num) {
		t.Error("file size is not correct after sync, ", info.Size(), num)
	}
}

func TestRead(t *testing.T) {
	var opt = BufferedFileOption{
		fileName,
		time.Second * 1,
		512,
	}

	var file, err = MakeBufferedFile(opt)
	defer func() {
		file.Close()
		os.Remove(fileName)
	}()

	if err != nil {
		t.Error(err)
	}

	if len(file.buffer) != opt.bufferSize {
		t.Log("buffer size is incorrect")
		t.Error(len(file.buffer))
	}

	num, err := file.Write(toWrite)
	fmt.Println(num)
	if num != len(toWrite) || err != nil {
		t.Log("write size error")
		t.Error(num, ", expect:", len(toWrite))
	}

	var read_buf = make([]byte, 1)
	num, err = file.ReadAt(read_buf, 0)
	if num != len(read_buf) || err != nil {
		t.Error("failed to read", err, num)
	}
	fmt.Println(read_buf)

	file.Write([]byte{1, 2, 3})
	fmt.Println(file.waterMark)
	num, err = file.ReadAt(read_buf, int64(len(toWrite)))
	if num != len(read_buf) || err != nil {
		t.Error("failed to read", err, num)
	}
	fmt.Println(read_buf)
}
