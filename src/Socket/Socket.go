package Socket

import "net"
import "time"
import cDebug "github.com/tj/go-debug"
import "sync"

var debugOut = cDebug.Debug("Socket")

type SocketClient struct {
	PackageChan chan Package
	con         *net.TCPConn
	GoingClose  chan bool
	rawChan     chan []byte
	closed      sync.Once
	closingLock sync.Mutex
}

// TODO: due with error
func (c *SocketClient) WriteRaw(data []byte) (int, error) {
	defer func() {
		if err := recover(); err != nil {
			c.Close()
		}
	}()
	c.closingLock.Lock()
	defer c.closingLock.Unlock()
	c.rawChan <- data
	return len(data), nil
}

func (c *SocketClient) sendPack(data []byte) (int, error) {
	return c.WriteRaw(protocolPack(data))
}

func (c *SocketClient) SendDataPack(data []byte) (int, error) {
	var header PackHeader = PackHeader{
		true,
		DATA,
	}
	var result, err = bufferToPack(data, header)
	if err != nil {
		return 0, err
	}
	return c.sendPack(result)
}

func (c *SocketClient) SendMessagePack(data []byte) (int, error) {
	var header PackHeader = PackHeader{
		true,
		MESSAGE,
	}
	var result, err = bufferToPack(data, header)
	if err != nil {
		return 0, err
	}
	return c.sendPack(result)
}

func (c *SocketClient) SendCommandPack(data []byte) (int, error) {
	var header PackHeader = PackHeader{
		true,
		COMMAND,
	}
	var result, err = bufferToPack(data, header)
	if err != nil {
		return 0, err
	}
	return c.sendPack(result)
}

func (c *SocketClient) SendManagerPack(data []byte) (int, error) {
	var header PackHeader = PackHeader{
		true,
		MANAGER,
	}
	var result, err = bufferToPack(data, header)
	if err != nil {
		return 0, err
	}
	return c.sendPack(result)
}

func AssamblePack(header PackHeader, data []byte) []byte {
	var result, err = bufferToPack(data, header)
	if err != nil {
		return make([]byte, 0)
	}
	return protocolPack(result)
}

func (c *SocketClient) Close() {
	defer func() { recover() }()
	c.closed.Do(func() {
		c.closingLock.Lock()
		defer c.closingLock.Unlock()
		close(c.GoingClose)
		close(c.PackageChan)
		close(c.rawChan)
		c.con.Close()
		debugOut("client closed")
	})
}

func writeLoop(client *SocketClient, con *net.TCPConn) {
	for {
		select {
		case _, _ = <-client.GoingClose:
			return
		case data, ok := <-client.rawChan:
			if !ok {
				debugOut("client rawChan already closed")
				client.Close()
				return
			}
			//client.con.SetWriteDeadline(time.Now().Add(20 * time.Second))
			_, err := client.con.Write(data)
			if err != nil {
				debugOut("cannot make write on client")
				client.Close()
				return
			}
			debugOut("wrote succeed")
		case <-time.After(60 * time.Second):
			debugOut("client write timeout")
			client.Close()
			return
		}
	}
}

func readLoop(client *SocketClient, con *net.TCPConn, reader *SocketReader) {
	for {
		select {
		case _, _ = <-client.GoingClose:
			return
		default:
			var buffer []byte = make([]byte, 128)
			outBytes, err := con.Read(buffer)
			//con.SetReadDeadline(time.Now().Add(20 * time.Second))
			if err != nil {
				client.Close()
				return
			}
			if outBytes == 0 {
				time.Sleep(1 * time.Second)
			} else {
				err = reader.OnData(buffer[0:outBytes])
				if err != nil {
					client.Close()
					return
				}
			}
		}

	}
}

func pubLoop(client *SocketClient, reader *SocketReader) {
	defer func() { recover() }()
	for {
		select {
		case _, _ = <-client.GoingClose:
			return
		case pkg, ok := <-reader.PackageChan:
			if !ok {
				return
			}
			// pipe Package into public scope
			client.closingLock.Lock()
			client.PackageChan <- pkg
			client.closingLock.Unlock()
		}
	}
}

func MakeSocketClient(con *net.TCPConn) *SocketClient {
	//con.SetReadDeadline(time.Now().Add(20 * time.Second))
	client := SocketClient{
		PackageChan: make(chan Package),
		con:         con,
		GoingClose:  make(chan bool),
		rawChan:     make(chan []byte, 40), // 40 arrays for each client
		closed:      sync.Once{},
		closingLock: sync.Mutex{},
	}
	reader := NewSocketReader()

	con.SetKeepAlive(true)
	con.SetNoDelay(true)
	con.SetLinger(10)

	go writeLoop(&client, con)
	go readLoop(&client, con, &reader)

	go pubLoop(&client, &reader)
	return &client
}
