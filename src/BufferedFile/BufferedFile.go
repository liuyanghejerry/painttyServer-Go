// BufferedFile provides a buffered file frontended by a []byte.
// And it can sync to file automatically
package BufferedFile

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

type BufferedFileOption struct {
	FileName   string
	WriteCycle time.Duration //  60*1000 // in milliseconds
	BufferSize int           //  1024*1024 // in bytes
}

type BufferedFile struct {
	option     BufferedFileOption
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
	file, err := os.OpenFile(f.option.FileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)

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

// FIXME: not working
func (f *BufferedFile) startWriteTimer() {
	go func() {
		for {
			select {
			case <-f.goingClose:
				f.goingClose <- true
				return
			default:
				time.Sleep(f.option.WriteCycle)
				fmt.Println("auto sync")
				if err := f.Sync(); err != nil {
					panic(err)
				}
			}
		}
	}()
}

func (f *BufferedFile) Sync() error {
	f.locker.Lock()
	defer f.locker.Unlock()
	var mark = atomic.LoadInt64(&f.waterMark)
	//fmt.Println("watermark read", mark)
	if mark < 1 {
		return nil
	}
	//fmt.Println("write to system file", mark)
	_, err := f.file.Write(f.buffer[0:mark])
	f.buffer = make([]byte, f.option.BufferSize) // optional, may re-use
	f.fileSize += mark
	//f.waterMark = 0
	atomic.StoreInt64(&f.waterMark, 0)
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

func (f *BufferedFile) Clear() error {
	f.locker.Lock()
	defer f.locker.Unlock()
	atomic.StoreInt64(&f.waterMark, 0)
	atomic.StoreInt64(&f.fileSize, 0)
	f.buffer = make([]byte, f.option.BufferSize) // optional, may re-use
	err := f.file.Truncate(0)
	return err
}

func (f *BufferedFile) Write(data []byte) (int64, error) {
	var length = int64(len(data))
	var left = int64(len(f.buffer)) - atomic.LoadInt64(&f.waterMark)
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
	f.locker.Lock()
	defer f.locker.Unlock()
	if length > int64(len(f.buffer)) {
		atomic.AddInt64(&f.wholeSize, length)
		f.fileSize += length
		l, err := f.file.Write(data)
		return int64(l), err
	}
	copy(f.buffer, data)
	//f.waterMark += length
	atomic.AddInt64(&f.waterMark, length)
	atomic.AddInt64(&f.wholeSize, length)
	//fmt.Println("watermark update", atomic.LoadInt64(&f.waterMark))
	return length, err
}

func (f *BufferedFile) ReadAt(data []byte, off int64) (int, error) {
	f.locker.Lock()
	defer f.locker.Unlock()
	var length = int64(len(data))
	var mark = atomic.LoadInt64(&f.waterMark)
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
		make([]byte, option.BufferSize),
		0,
		0,
		0,
		nil,
		make(chan bool, 1),
		sync.Mutex{},
	}

	if err := bufFile.openForRead(); err != nil {
		return bufFile, err
	}

	if err := bufFile.fetchFileSize(); err != nil {
		return bufFile, err
	}
	atomic.StoreInt64(&bufFile.wholeSize, bufFile.fileSize)
	//bufFile.startWriteTimer()
	return bufFile, nil
}
