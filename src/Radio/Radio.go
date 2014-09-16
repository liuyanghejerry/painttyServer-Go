package Radio

import "time"

import "fmt"
import "Socket"
import "BufferedFile"
import "sync"

type RadioTaskList struct {
	tasks  []RadioChunk
	locker sync.Mutex
}

func (r *RadioTaskList) Tasks() *[]RadioChunk {
	r.locker.Lock()
	defer r.locker.Unlock()
	return &(r.tasks)
}

func (r *RadioTaskList) Length() int {
	r.locker.Lock()
	defer r.locker.Unlock()
	return len(r.tasks)
}

func (r *RadioTaskList) Append(chunks []RadioChunk) {
	r.locker.Lock()
	defer r.locker.Unlock()
	r.tasks = append(r.tasks, chunks...)
}

func (r *RadioTaskList) PushBack(chunks []RadioChunk) {
	r.Append(chunks)
}

func (r *RadioTaskList) PopBack() RadioChunk {
	r.locker.Lock()
	defer r.locker.Unlock()
	var bottomItem = r.tasks[len(r.tasks)-1]
	r.tasks = r.tasks[:len(r.tasks)-1]
	return bottomItem
}

func (r *RadioTaskList) PopFront() RadioChunk {
	r.locker.Lock()
	defer r.locker.Unlock()
	var item = r.tasks[0]
	r.tasks = r.tasks[1:len(r.tasks)]
	return item
}

func (r *RadioTaskList) PushFront(chunk RadioChunk) {
	r.locker.Lock()
	defer r.locker.Unlock()
	r.tasks = append(r.tasks, chunk)
	copy(r.tasks[1:], r.tasks[0:])
	r.tasks[0] = chunk
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
	list      *RadioTaskList
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
		v.list = &RadioTaskList{
			make([]RadioChunk, 0, 100),
			sync.Mutex{},
		}
	}
	if err := r.file.Clear(); err != nil {
		panic(err)
	}
	r.signature = genSignature()
}

func (r *Radio) AddClient(client *Socket.SocketClient, start, length int64) {
	r.locker.Lock()
	defer r.locker.Unlock()

	var list = &RadioTaskList{
		make([]RadioChunk, 0),
		sync.Mutex{},
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
		//list.tasks = append(list.tasks, chunks...)
		list.Append(chunks)
		fmt.Println("tasks assigned", list.Tasks())
	}
	var radioClient = RadioClient{
		client:    client,
		sendChan:  make(chan RAMChunk),
		writeChan: make(chan FileChunk),
		list:      list,
	}
	//fmt.Println("init tasks", radioClient.list)

	r.clients[client] = &radioClient

	go func() {
		for {
			select {
			case <-client.GoingClose:
				client.GoingClose <- true
				return
			case chunk := <-radioClient.sendChan:
				appendToPendings(chunk, radioClient.list)
				break
			case chunk := <-radioClient.writeChan:
				fmt.Println("write chan happened")
				appendToPendings(chunk, radioClient.list)
				break
			default:
				time.Sleep(time.Millisecond * 300)
				fetchAndSend(client, radioClient.list, r.file)
			}

		}
	}()
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
	wrote, err := r.file.Write(data)
	fmt.Println("wrote", wrote, "into radio")
	if err != nil {
		panic(err)
	}
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
