// BufferedFile provides a buffered file frontended by a []byte.
// And it can sync to file automatically
package BufferedFile

import (
	"errors"
	cDebug "github.com/tj/go-debug"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

var debugOut = cDebug.Debug("BufferedFile")

type BufferedFileOption struct {
	FileName   string
	WriteCycle time.Duration //  60*1000 // in milliseconds
	BufferSize int32         //  1024*50 // in bytes
}

type BufferedFile struct {
	option     *BufferedFileOption
	buffer     []byte
	waterMark  int64
	fileSize   int64
	wholeSize  int64
	file       *os.File
	goingClose chan bool
	locker     sync.Mutex
}

// Returns the size of file and buffer
func (f *BufferedFile) WholeSize() int64 {
	return atomic.LoadInt64(&f.wholeSize)
}

func (f *BufferedFile) openForRead() error {
	file, err := os.OpenFile(f.option.FileName, os.O_RDWR|os.O_CREATE, 0666)

	if err != nil {
		return err
	}

	f.file = file
	file.Seek(0, 2)
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

func (f *BufferedFile) startWriteTimer() {
	go func() {
		for {
			select {
			case _, _ = <-f.goingClose:
				return
			case <-time.After(f.option.WriteCycle):
				if err := f.Sync(); err != nil {
					panic(err)
				}
			}
		}
	}()
}

func (f *BufferedFile) Sync() error {
	f.locker.Lock()
	defer func() {
		f.file.Close()
		f.openForRead()
		f.locker.Unlock()
	}()
	var mark = atomic.LoadInt64(&f.waterMark)
	//debugOut("watermark read", mark)
	if mark < 1 {
		return nil
	}
	//debugOut("write to system file", mark)
	_, err := f.file.Write(f.buffer[0:mark])
	//f.buffer = make([]byte, f.option.BufferSize) // // FIXME: seems leaking
	atomic.AddInt64(&f.fileSize, mark)
	//f.waterMark = 0
	atomic.StoreInt64(&f.waterMark, 0)
	return err
}

func (f *BufferedFile) innerSync() error {
	defer func() {
		f.file.Close()
		f.openForRead()
	}()
	var mark = atomic.LoadInt64(&f.waterMark)
	//debugOut("watermark read", mark)
	if mark < 1 {
		return nil
	}
	//debugOut("write to system file", mark)
	_, err := f.file.Write(f.buffer[0:mark])
	//f.buffer = make([]byte, f.option.BufferSize) // FIXME: seems leaking
	atomic.AddInt64(&f.fileSize, mark)
	//f.waterMark = 0
	atomic.StoreInt64(&f.waterMark, 0)
	return err
}

func (f *BufferedFile) Close() error {
	close(f.goingClose)
	err := f.Sync()
	if err != nil {
		return err
	}
	err = f.file.Close()
	return err
}

func (f *BufferedFile) Clear() error {
	f.locker.Lock()
	defer f.locker.Unlock()
	atomic.StoreInt64(&f.wholeSize, 0)
	atomic.StoreInt64(&f.waterMark, 0)
	atomic.StoreInt64(&f.fileSize, 0)
	//f.buffer = make([]byte, f.option.BufferSize) // FIXME: same reason
	f.file.Seek(0, 0)
	if err := f.file.Truncate(0); err != nil {
		return err
	}
	return nil
}

func (f *BufferedFile) Remove() {
	os.Remove(f.option.FileName)
}

func (f *BufferedFile) Write(data []byte) (int64, error) {
	f.locker.Lock()
	defer f.locker.Unlock()
	var length = int64(len(data))
	var left = int64(len(f.buffer)) - atomic.LoadInt64(&f.waterMark)
	var err error = nil
	// left room not enough
	if left <= length {
		err = f.innerSync()
		if err != nil {
			return 0, err
		}
		// sync updated waterMark and buffer
		left = int64(len(f.buffer)) - atomic.LoadInt64(&f.waterMark)
	}
	// buffer cannot contain such big write
	// directly write into file
	// since we've already clear the buffer
	if length > int64(len(f.buffer)) {
		atomic.AddInt64(&f.wholeSize, length)
		atomic.AddInt64(&f.fileSize, length)
		l, err := f.file.Write(data)
		return int64(l), err
	}
	for i, j := atomic.LoadInt64(&f.waterMark), int64(0); j < int64(len(data)) && i < int64(len(f.buffer)); i, j = i+1, j+1 {
		f.buffer[i] = data[j]
	}
	atomic.AddInt64(&f.waterMark, length)
	atomic.AddInt64(&f.wholeSize, length)
	return length, err
}

func (f *BufferedFile) ReadAt(data []byte, off int64) (int64, error) {
	f.locker.Lock()
	defer f.locker.Unlock()
	var fileSize = atomic.LoadInt64(&f.fileSize)
	var length = int64(len(data))
	var mark = atomic.LoadInt64(&f.waterMark)
	var err error = nil
	debugOut("fileSize: %d, waterMark: %d, whileSize: %d, readOff: %d, readLen: %d",
		fileSize, mark, f.wholeSize, off, length)
	if off+length > fileSize+mark {
		return 0, errors.New("Cannot read so much")
	}
	// all in file
	if off+length <= fileSize && fileSize != 0 {
		debugOut("all in file")
		l, e := f.file.ReadAt(data, off)
		return int64(l), e
	}

	// all in buffer
	if off > fileSize {
		debugOut("all in buffer")
		start := off - fileSize
		num := copy(data, f.buffer[start:start+length])
		return int64(num), nil
	}

	// half in buffer, and the other half in file
	// read file first
	debugOut("half, half")
	var file_buf = make([]byte, fileSize-off)
	_, err = f.file.ReadAt(file_buf, off)
	// copy bytes in buffer then
	var buffer_buf = make([]byte, off+length-fileSize)
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

	return int64(len(data)), err
}

func MakeBufferedFile(option *BufferedFileOption) (*BufferedFile, error) {
	var bufFile = BufferedFile{
		option,
		make([]byte, option.BufferSize),
		0,
		0,
		0,
		nil,
		make(chan bool),
		sync.Mutex{},
	}

	if err := bufFile.openForRead(); err != nil {
		return &bufFile, err
	}

	if err := bufFile.fetchFileSize(); err != nil {
		return &bufFile, err
	}
	atomic.StoreInt64(&bufFile.wholeSize, bufFile.fileSize)
	bufFile.startWriteTimer()
	return &bufFile, nil
}
