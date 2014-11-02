package Radio

import "time"
import "Socket"
import "BufferedFile"
import "sync"
import cDebug "github.com/tj/go-debug"

var debugOut = cDebug.Debug("Radio")

type RadioTaskList struct {
	tasks  []RadioChunk
	locker sync.Mutex
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

func (r *Radio) Remove() {
	r.locker.Lock()
	defer r.locker.Unlock()
	r.file.Close()
	defer r.file.Remove()
}

func (r *Radio) Signature() string {
	return r.signature
}

func (r *Radio) Prune() string {
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
	r.signature = genArchiveSign(r.signature)
	return r.signature
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
		debugOut("tasks assigned", list.Tasks())
	}
	var radioClient = RadioClient{
		client:    client,
		sendChan:  make(chan RAMChunk),
		writeChan: make(chan FileChunk),
		list:      list,
	}
	//debugOut("init tasks", radioClient.list)

	r.clients[client] = &radioClient

	go r.processClient(client, &radioClient)
}

func (r *Radio) processClient(client *Socket.SocketClient, radioClient *RadioClient) {
	for {
		select {
		case _, _ = <-client.GoingClose:
			r.RemoveClient(client)
			return
		case chunk, ok := <-radioClient.sendChan:
			if ok {
				debugOut("send chan happened")
				appendToPendings(chunk, radioClient.list)
			} else {
				debugOut("send chan miss-matched")
				r.RemoveClient(client)
				return
			}
		case chunk, ok := <-radioClient.writeChan:
			if ok {
				debugOut("write chan happened")
				appendToPendings(chunk, radioClient.list)
			} else {
				debugOut("write chan miss-matched")
				r.RemoveClient(client)
				return
			}
		case <-time.After(time.Millisecond * 100):
			fetchAndSend(client, radioClient.list, r.file)
			//default:
			//	fetchAndSend(client, radioClient.list, r.file)
		}

	}
}

func (r *Radio) RemoveClient(client *Socket.SocketClient) {
	r.locker.Lock()
	defer r.locker.Unlock()
	debugOut("remove client from radio")
	delete(r.clients, client)
}

func (r *Radio) RemoveAllClients() {
	r.locker.Lock()
	defer r.locker.Unlock()
	debugOut("remove client from radio")
	r.clients = make(map[*Socket.SocketClient]*RadioClient)
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
	//go func() {
	select {
	case cli.sendChan <- RAMChunk{data}:
	case <-time.After(time.Second * 10):
		r.RemoveClient(client)
		debugOut("sendChan failed in singleSend")
	}
	//}()
}

func send_helper(
	client *Socket.SocketClient,
	cli *RadioClient,
	data []byte,
	r *Radio) {
	defer func() { recover() }()
	select {
	case cli.sendChan <- RAMChunk{data}:
	case <-time.After(time.Second * 10):
		r.RemoveClient(client)
		debugOut("sendChan failed in send")
	}
}

// Send expected Buffer that send to every Client but doesn't record.
func (r *Radio) send(data []byte) {
	r.locker.Lock()
	defer r.locker.Unlock()
	for client, cli := range r.clients {
		go send_helper(client, cli, data, r)
	}
}

func write_helper(
	client *Socket.SocketClient,
	cli *RadioClient,
	data []byte,
	r *Radio,
	oldPos int64) {
	defer func() { recover() }() // in case cli.writeChan is closed
	select {
	case cli.writeChan <- FileChunk{
		Start:  oldPos,
		Length: int64(len(data)),
	}:
	case <-time.After(time.Second * 10):
		r.RemoveClient(client)
		debugOut("writeChan failed in write")
	}
}

// Write expected Buffer that send to every Client and record data.
func (r *Radio) write(data []byte) {
	var oldPos = r.file.WholeSize()
	wrote, err := r.file.Write(data)
	debugOut("wrote", wrote, "into radio")
	if err != nil {
		panic(err)
	}

	r.locker.Lock()
	defer r.locker.Unlock()
	for client, cli := range r.clients {
		debugOut("published")
		go write_helper(client, cli, data, r, oldPos)
	}
}

func (r *Radio) run() {
	for {
		select {
		case _, _ = <-r.GoingClose:
			return
		case part, ok := <-r.SendChan:
			if !ok {
				return
			}
			r.send(part.Data)
		case part, ok := <-r.SingleSendChan:
			if !ok {
				return
			}
			r.singleSend(part.Data, part.Client)
		case part, ok := <-r.WriteChan:
			if !ok {
				return
			}
			r.write(part.Data)
		}
	}
}

func MakeRadio(fileName, sign string) (*Radio, error) {
	var file, err = BufferedFile.MakeBufferedFile(
		&BufferedFile.BufferedFileOption{
			fileName,
			time.Second * 20,
			1024 * 50,
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
		signature:      sign,
	}
	go radio.run()
	return radio, nil
}
