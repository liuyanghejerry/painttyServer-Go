package Socket

import "net"
import "time"
import "log"
import "sync"

type SocketCloseCallback func()

type SocketClient struct {
	readLock              sync.Mutex
	writeLock             sync.Mutex
	con                   *net.TCPConn
	closeFlag             sync.Once
	packageChan           chan Package
	closeCallbackList     []SocketCloseCallback
	closeCallbackListLock sync.Mutex
}

func (c *SocketClient) RegisterCloseCallback(callback SocketCloseCallback) {
	c.closeCallbackListLock.Lock()
	defer c.closeCallbackListLock.Unlock()
	c.closeCallbackList = append(c.closeCallbackList, callback)
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
		close(c.packageChan)
		c.closeCallbackListLock.Lock()
		defer c.closeCallbackListLock.Unlock()
		for i := 0; i < len(c.closeCallbackList); i++ {
			c.closeCallbackList[i]()
		}
		c.closeCallbackList = make([]SocketCloseCallback, 0)
	})
}

func (c *SocketClient) RunReadLoop(reader *SocketReader) {
	reader.RegisterHandler(func(pkg Package) {
		defer func() {
			if x := recover(); x != nil {
				log.Printf("recovered panic: %v", x)
			}
		}()
		c.packageChan <- pkg
	})
	for {
		buffer := make([]byte, 128)
		c.readLock.Lock()
		// c.con.SetReadDeadline(<-time.After(60 * time.Second))
		outBytes, err := c.con.Read(buffer)
		c.readLock.Unlock()
		if err != nil {
			reader.UnregisterHandler()
			c.Close()
			break
		}
		if outBytes == 0 {
			time.Sleep(1 * time.Second)
			continue
		}
		err = reader.OnData(buffer[0:outBytes])
		if err != nil {
			log.Println(err)
			reader.UnregisterHandler()
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

	go client.RunReadLoop(reader)
	return &client
}
