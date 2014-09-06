package Radio

import "time"

//import "fmt"
import "Socket"
import "BufferedFile"
import "sync"

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

type RadioClient struct {
	client    *Socket.SocketClient
	sendChan  chan RAMChunk
	writeChan chan FileChunk
	list      RadioTaskList
}

type RadioSendPart struct {
	Data []byte
}

type RadioSingleSendPart struct {
	Data   []byte
	Client *Socket.SocketClient
}

type Radio struct {
	clients        map[*Socket.SocketClient]*RadioClient
	file           *BufferedFile.BufferedFile
	GoingClose     chan bool
	SingleSendChan chan RadioSingleSendPart
	SendChan       chan RadioSendPart
	WriteChan      chan RadioSendPart
	signature      string
	locker         sync.RWMutex
}

func (r *Radio) Close() {
	r.GoingClose <- true
}

func (r *Radio) Signature() string {
	return r.signature
}

func (r *Radio) Prune() {
	r.locker.Lock()
	defer r.locker.Unlock()
	for _, v := range r.clients {
		v.list = RadioTaskList{
			make([]RadioChunk, 0, 100),
		}
	}
	if err := r.file.Clear(); err != nil {
		panic(err)
	}
	r.signature = genSignature()
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
	var radioClient = RadioClient{
		client:    client,
		sendChan:  make(chan RAMChunk),
		writeChan: make(chan FileChunk),
		list: RadioTaskList{
			make([]RadioChunk, 0, 100),
		},
	}
	//fmt.Println("init tasks", radioClient.list)
	r.locker.Lock()
	r.clients[client] = &radioClient
	r.locker.Unlock()
	for {
		select {
		case <-client.GoingClose:
			client.GoingClose <- true
			return
		case chunk := <-radioClient.sendChan:
			list := radioClient.list
			radioClient.list = appendToPendings(chunk, list)
			break
		case chunk := <-radioClient.writeChan:
			list := radioClient.list
			radioClient.list = appendToPendings(chunk, list)
			break
		default:
			time.Sleep(time.Millisecond * 1500)
			list := radioClient.list
			list = fetchAndSend(client, list, r.file)
			radioClient.list = list
		}

	}
}

func (r *Radio) FileSize() int64 {
	return r.file.WholeSize()
}

// SingleSend expected Buffer that send to one specific Client but doesn't record.
func (r *Radio) singleSend(data []byte, client *Socket.SocketClient) {
	r.locker.RLock()
	cli, ok := r.clients[client]
	r.locker.RUnlock()
	if !ok {
		return
	}
	cli.sendChan <- RAMChunk{
		data,
	}
}

// Send expected Buffer that send to every Client but doesn't record.
func (r *Radio) send(data []byte) {
	for _, cli := range r.clientList() {
		cli.sendChan <- RAMChunk{
			data,
		}
	}
}

// Write expected Buffer that send to every Client and record data.
func (r *Radio) write(data []byte) {
	var oldPos = r.file.WholeSize()
	_, err := r.file.Write(data)
	if err != nil {
		panic(err)
	}
	//fmt.Println("file is", r.file.WholeSize())
	for _, cli := range r.clientList() {
		cli.writeChan <- FileChunk{
			Start:  oldPos,
			Length: int64(len(data)),
		}
	}
}

func (r *Radio) clientList() []*RadioClient {
	list := make([]*RadioClient, 0)
	r.locker.RLock()
	defer r.locker.RUnlock()
	for _, cli := range r.clients {
		list = append(list, cli)
	}
	return list
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

func MakeRadio(fileName string) (*Radio, error) {
	var file, err = BufferedFile.MakeBufferedFile(BufferedFile.BufferedFileOption{
		fileName,
		time.Second * 3,
		1024 * 100,
	})
	if err != nil {
		return &Radio{}, err
	}
	var radio = Radio{
		clients:        make(map[*Socket.SocketClient]*RadioClient),
		file:           file,
		GoingClose:     make(chan bool),
		SingleSendChan: make(chan RadioSingleSendPart),
		SendChan:       make(chan RadioSendPart),
		WriteChan:      make(chan RadioSendPart),
		locker:         sync.RWMutex{},
		signature:      genSignature(), // TODO: recovery
	}
	go radio.run()
	return &radio, nil
}
