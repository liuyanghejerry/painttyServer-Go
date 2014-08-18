package Radio

import "time"
import "Socket"
import "BufferedFile"

type RadioTaskList struct {
	tasks []RadioChunk
}

type RadioChunk interface {
	special()
}

type RAMChunk struct {
	Data []byte
}

type FileChunk struct {
	Start  int64
	Length int64
}

// ensure interface
func (c RAMChunk) special() {
	//
}

// ensure interface
func (c FileChunk) special() {
	//
}

type Radio struct {
	clients    map[*Socket.SocketClient]RadioTaskList
	file       BufferedFile.BufferedFile
	GoingClose chan bool
}

func (r *Radio) Close() {
	r.GoingClose <- true
}

func (r *Radio) AddClient(client *Socket.SocketClient, start, length int64) {
	var list = RadioTaskList{}
	var fileSize = r.file.WholeSize()
	var startPos, chunkSize int64
	if start > fileSize {
		startPos = fileSize
	} else {
		startPos = start
	}
	if startPos+length > fileSize {
		chunkSize = length
	} else {
		chunkSize = fileSize - startPos
	}
	if chunkSize != 0 {
		var chunks = splitChunk(FileChunk{
			Start:  startPos,
			Length: chunkSize,
		})
		// appending to list
		list.tasks = append(list.tasks, chunks...)
	}
	r.clients[client] = list
}

func MakeRadio(fileName string) (Radio, error) {
	var file, err = BufferedFile.MakeBufferedFile(BufferedFile.BufferedFileOption{
		fileName,
		time.Second * 3,
		1024 * 1024,
	})
	if err != nil {
		return Radio{}, err
	}
	var radio = Radio{
		file:       file,
		GoingClose: make(chan bool, 1),
	}

	return radio, nil
}
