package Radio

import "time"

//import "fmt"
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

type RadioSendPart struct {
	Data []byte
}

type RadioSingleSendPart struct {
	Data   []byte
	Client *Socket.SocketClient
}

type Radio struct {
	clients        map[*Socket.SocketClient]RadioTaskList
	file           BufferedFile.BufferedFile
	GoingClose     chan bool
	SingleSendChan chan RadioSingleSendPart
	SendChan       chan RadioSendPart
	WriteChan      chan RadioSendPart
}

func (r *Radio) Close() {
	r.GoingClose <- true
}

func (r *Radio) AddClient(client *Socket.SocketClient, start, length int64) {
	var list = RadioTaskList{
		make([]RadioChunk, 0, 100),
	}
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

func (r *Radio) FileSize() int64 {
	return r.file.WholeSize()
}

// SingleSend expected Buffer that send to one specific Client but doesn't record.
func (r *Radio) singleSend(data []byte, client *Socket.SocketClient) {
	value, ok := r.clients[client]
	if !ok {
		return
	}
	value = appendToPendings(RAMChunk{
		data,
	}, value)
}

// Send expected Buffer that send to every Client but doesn't record.
func (r *Radio) send(data []byte) {
	for _, value := range r.clients {
		//fmt.Println("Key:", key, "Value:", value)
		value = appendToPendings(RAMChunk{
			data,
		}, value)
	}
}

// Write expected Buffer that send to every Client and record data.
func (r *Radio) write(data []byte) {
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

func (r *Radio) run() {
	for {
		select {
		case <-r.GoingClose:
			r.GoingClose <- true
			return
		case part := <-r.SendChan:
			r.send(part.Data)
			break
		case part := <-r.SingleSendChan:
			r.singleSend(part.Data, part.Client)
			break
		case part := <-r.WriteChan:
			r.write(part.Data)
			break
		}
	}
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
		clients:        make(map[*Socket.SocketClient]RadioTaskList),
		file:           file,
		GoingClose:     make(chan bool),
		SingleSendChan: make(chan RadioSingleSendPart),
		SendChan:       make(chan RadioSendPart),
		WriteChan:      make(chan RadioSendPart),
	}
	go radio.run()
	return radio, nil
}
