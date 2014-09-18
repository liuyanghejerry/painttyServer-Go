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
	locker         sync.Mutex
}

func (r *Radio) Close() {
	close(r.GoingClose)
	close(r.SingleSendChan)
	close(r.SendChan)
	close(r.WriteChan)
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
			case _, _ = <-client.GoingClose:
				fmt.Println("auto remove closed client from radio")
				r.locker.Lock()
				delete(r.clients, client)
				r.locker.Unlock()
				return
			case chunk, ok := <-radioClient.sendChan:
				if ok {
					appendToPendings(chunk, radioClient.list)
				} else {
					return
				}
			case chunk, ok := <-radioClient.writeChan:
				if ok {
					fmt.Println("write chan happened")
					appendToPendings(chunk, radioClient.list)
				} else {
					return
				}
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
	r.locker.Lock()
	defer r.locker.Unlock()

	cli, ok := r.clients[client]
	if !ok {
		return
	}
	go func() {
		select {
		case cli.sendChan <- RAMChunk{data}:
		case <-time.After(time.Second * 10):
			fmt.Println("sendChan failed in singleSend")
		}
	}()
}

// Send expected Buffer that send to every Client but doesn't record.
func (r *Radio) send(data []byte) {
	for _, cli := range r.clientList() {
		go func() {
			select {
			case cli.sendChan <- RAMChunk{data}:
			case <-time.After(time.Second * 10):
				fmt.Println("sendChan failed in send")
			}
		}()

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
		fmt.Println("published")
		go func() {
			select {
			case cli.writeChan <- FileChunk{
				Start:  oldPos,
				Length: int64(len(data)),
			}:
			case <-time.After(time.Second * 10):
				fmt.Println("writeChan failed in write")
			}
		}()
	}
}

func (r *Radio) clientList() []*RadioClient {
	r.locker.Lock()
	defer r.locker.Unlock()

	list := make([]*RadioClient, 0)
	for _, cli := range r.clients {
		list = append(list, cli)
	}
	return list
}

func (r *Radio) run() {
	for {
		select {
		case _, _ = <-r.GoingClose:
			return
		case part, ok := <-r.SendChan:
			if ok {
				r.send(part.Data)
			} else {
				return
			}
		case part, ok := <-r.SingleSendChan:
			if ok {
				r.singleSend(part.Data, part.Client)
			} else {
				return
			}
		case part, ok := <-r.WriteChan:
			if ok {
				r.write(part.Data)
			} else {
				return
			}
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
	var radio = &Radio{
		clients:        make(map[*Socket.SocketClient]*RadioClient),
		file:           file,
		GoingClose:     make(chan bool),
		SingleSendChan: make(chan RadioSingleSendPart),
		SendChan:       make(chan RadioSendPart),
		WriteChan:      make(chan RadioSendPart),
		locker:         sync.Mutex{},
		signature:      genSignature(), // TODO: recovery
	}
	go radio.run()
	return radio, nil
}
