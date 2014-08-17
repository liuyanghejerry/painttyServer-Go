package BufferedFile

import "os"
import "time"
import "errors"

//import "fmt"

type BufferedFileOption struct {
	fileName   string
	writeCycle time.Duration //  60*1000 // in milliseconds
	bufferSize int           //  1024*1024 // in bytes
}

type BufferedFile struct {
	option     BufferedFileOption
	buffer     []byte
	waterMark  int
	fileSize   int64
	wholeSize  int64
	file       *os.File
	goingClose chan bool
}

func (f *BufferedFile) openForRead() error {
	file, err := os.OpenFile(f.option.fileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)

	if err != nil {
		return err
	}

	f.file = file
	return nil
}

func (f *BufferedFile) fetchFileSize() error {
	fi, err := f.file.Stat()
	if err != nil {
		return err
	}

	f.fileSize = fi.Size()
	return err
}

func (f *BufferedFile) startWriteTimer() error {
	go func() {
		for {
			select {
			case <-f.goingClose:
				f.goingClose <- true
				return
			default:
				if err := f.Sync(); err != nil {
					panic(err)
				}
				time.Sleep(f.option.writeCycle * time.Millisecond)
			}
		}
	}()
	return nil
}

func (f *BufferedFile) Sync() error {
	_, err := f.file.Write(f.buffer[0:f.waterMark])
	f.buffer = make([]byte, f.option.bufferSize) // optional, may re-use
	f.fileSize += int64(f.waterMark)
	f.waterMark = 0
	return err
}

func (f *BufferedFile) Close() error {
	err := f.Sync()
	if err != nil {
		return err
	}
	err = f.file.Close()
	return err
}

func (f *BufferedFile) Write(data []byte) (int, error) {
	var length = len(data)
	var left = len(f.buffer) - f.waterMark
	var err error = nil
	// left room not enough
	if left <= length {
		err = f.Sync()
		if err != nil {
			return 0, err
		}
	}
	// buffer cannot contain such big write
	// directly write into file
	// since we've already clear the buffer
	if length > len(f.buffer) {
		f.wholeSize += int64(length)
		f.fileSize += int64(length)
		return f.file.Write(data)
	}
	copy(f.buffer, data)
	f.waterMark += length
	f.wholeSize += int64(length)
	return length, err
}

// FIXME: read should have a clear pos
func (f *BufferedFile) Read(data []byte) (int, error) {
	var length = len(data)
	var err error = nil
	// none in buffer
	if f.waterMark == 0 {
		return f.file.Read(data)
	}

	// all in buffer
	if length <= f.waterMark {
		num := copy(data, f.buffer)
		return num, nil
	}
	// half in buffer, and the other half in file
	// copy bytes in buffer first
	for i := 0; i < f.waterMark; i++ {
		data[i] = f.buffer[i]
	}
	// all in one?
	_, err = f.file.Read(data[f.waterMark:])

	return length, err
}

func (f *BufferedFile) ReadAt(data []byte, off int64) (int, error) {
	var length = int64(len(data))
	var mark = int64(f.waterMark)
	var err error = nil
	if length > f.fileSize+mark {
		return 0, errors.New("Cannot read so much")
	}
	// all in file
	if off+length <= f.fileSize {
		return f.file.ReadAt(data, off)
	}

	// all in buffer
	if off+length <= mark {
		num := copy(data, f.buffer)
		return num, nil
	}

	// half in buffer, and the other half in file
	// read file first
	var file_buf = make([]byte, f.fileSize-off)
	_, err = f.file.ReadAt(file_buf, off)
	// copy bytes in buffer then
	var buffer_buf = make([]byte, off+length-f.fileSize)
	for i := 0; i < len(buffer_buf); i++ {
		buffer_buf[i] = f.buffer[i]
	}
	// combine two parts
	for i := 0; i < len(file_buf); i++ {
		data[i] = file_buf[i]
	}
	for i, pre := 0, len(file_buf); i < len(buffer_buf); i++ {
		data[pre+i] = buffer_buf[i]
	}

	return len(data), err
}

func MakeBufferedFile(option BufferedFileOption) (BufferedFile, error) {
	var bufFile = BufferedFile{
		option,
		make([]byte, option.bufferSize),
		0,
		0,
		0,
		nil,
		make(chan bool, 1),
	}

	if err := bufFile.openForRead(); err != nil {
		return bufFile, err
	}

	if err := bufFile.fetchFileSize(); err != nil {
		return bufFile, err
	}
	bufFile.wholeSize = bufFile.fileSize
	return bufFile, nil
}
