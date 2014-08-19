package Radio

import "time"
import "Socket"
import "BufferedFile"

type RadioTaskList struct {
	tasks []RadioChunk
}

type RadioChunk interface {
	special() // in case others may mis-use this interface
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

// SingleSend expected Buffer that send to one specific Client but doesn't record.
func (r *Radio) SingleSend(data []byte, client *Socket.SocketClient) {
	value, ok := r.clients[client]
	if !ok {
		return
	}
	value = appendToPendings(RAMChunk{
		data,
	}, value)
}

// Send expected Buffer that send to every Client but doesn't record.
func (r *Radio) Send(data []byte) {
	for _, value := range r.clients {
		//fmt.Println("Key:", key, "Value:", value)
		value = appendToPendings(RAMChunk{
			data,
		}, value)
	}
}

// Write expected Buffer that send to every Client and record data.
func (r *Radio) Write(data []byte) {
	var oldPos = r.file.WholeSize()
	r.file.Write(data)
	for _, value := range r.clients {
		//fmt.Println("Key:", key, "Value:", value)
		value = appendToPendings(FileChunk{
			Start:  oldPos,
			Length: int64(len(data)),
		}, value)
	}
}

func (r *Radio) fetchAndSend(client *Socket.SocketClient) {
	list, ok := r.clients[client]
	if !ok {
		return
	}

	list = fetchAndSend(client, list, r.file)
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
