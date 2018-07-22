package Socket

import "net"
import "time"
import cDebug "github.com/visionmedia/go-debug"
import "sync"
import "sync/atomic"

var debugOut = cDebug.Debug("Socket")

type SocketClient struct {
	readLock    sync.Mutex
	writeLock   sync.Mutex
	con         *net.TCPConn
	closeFlag   sync.Once
	closed      int32
	packageChan chan Package
}

func (c *SocketClient) GetPackageChan() <-chan Package {
	return c.packageChan
}

func (c *SocketClient) WriteRaw(data []byte) (int, error) {
	c.writeLock.Lock()
	defer c.writeLock.Unlock()
	// c.con.SetWriteDeadline(<-time.After(60 * time.Second))
	return c.con.Write(data)
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
	c.closeFlag.Do(func() {
		c.con.Close()
		atomic.StoreInt32(&c.closed, 1)
		close(c.packageChan)
	})
}

func (c *SocketClient) IsClosed() bool {
	return atomic.LoadInt32(&c.closed) == 1
}

func (c *SocketClient) RunReadLoop(reader *SocketReader) {
	go func() {
		for {
			pkg, ok := <-reader.PackageChan
			if !ok {
				return
			}
			if c.IsClosed() {
				return
			}
			// pipe Package into public scope
			c.packageChan <- pkg
		}
	}()
	for {
		buffer := make([]byte, 128)
		c.readLock.Lock()
		// c.con.SetReadDeadline(<-time.After(60 * time.Second))
		outBytes, err := c.con.Read(buffer)
		c.readLock.Unlock()
		if err != nil {
			c.Close()
			break
		}
		if outBytes == 0 {
			time.Sleep(1 * time.Second)
			continue
		}
		err = reader.OnData(buffer[0:outBytes])
		if err != nil {
			debugOut("socket is closed")
			c.Close()
			break
		}
	}
}

func MakeSocketClient(con *net.TCPConn) *SocketClient {
	client := SocketClient{
		con:         con,
		closeFlag:   sync.Once{},
		packageChan: make(chan Package),
	}
	reader := NewSocketReader()

	con.SetKeepAlive(true)
	con.SetNoDelay(true)
	con.SetLinger(10)

	go client.RunReadLoop(&reader)
	return &client
}
