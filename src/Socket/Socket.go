package Socket

import "net"
import "time"
import "fmt"
import "sync"

type SocketClient struct {
	PackageChan chan Package
	con         *net.TCPConn
	GoingClose  chan bool
	rawChan     chan []byte
	closed      sync.Once
}

// TODO: due with error
func (c *SocketClient) WriteRaw(data []byte) (int, error) {
	defer func() { recover() }()
	fmt.Println("delivering")
	c.rawChan <- data
	fmt.Println("delivered")
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

func (c *SocketClient) Close() {
	defer func() { recover() }()
	c.closed.Do(func() {
		close(c.GoingClose)
		close(c.PackageChan)
		close(c.rawChan)
		c.con.Close()
		fmt.Println("client closed")
	})
}

func MakeSocketClient(con *net.TCPConn) *SocketClient {
	con.SetReadDeadline(time.Now().Add(20 * time.Second))
	client := SocketClient{
		make(chan Package),
		con,
		make(chan bool),
		make(chan []byte),
		sync.Once{},
	}
	reader := NewSocketReader()

	go func() {
		for {
			select {
			case _, _ = <-client.GoingClose:
				return
			case data, ok := <-client.rawChan:
				if !ok {
					fmt.Println("client rawChan already closed")
					client.Close()
					return
				}
				client.con.SetWriteDeadline(time.Now().Add(20 * time.Second))
				_, err := client.con.Write(data)
				if err != nil {
					fmt.Println("cannot make write on client")
					client.Close()
					return
				}
			case <-time.After(20 * time.Second):
				fmt.Println("client write timeout")
				client.Close()
			}
		}
	}()

	go func() {
		for {
			select {
			case _, _ = <-client.GoingClose:
				return
			default:
				var buffer []byte = make([]byte, 128)
				outBytes, err := con.Read(buffer)
				con.SetReadDeadline(time.Now().Add(20 * time.Second))
				if err != nil {
					client.Close()
					return
				}
				if outBytes == 0 {
					time.Sleep(1 * time.Second)
				} else {
					reader.OnData(buffer[0:outBytes])
				}
			}

		}
	}()
	go func() {
		for {
			select {
			case _, _ = <-client.GoingClose:
				return
			case pkg, ok := <-reader.PackageChan:
				if !ok {
					return
				}
				func() {
					defer func() { recover() }()
					// pipe Package into public scope
					client.PackageChan <- pkg
				}()
			}
		}
	}()
	return &client
}
