package Socket

import "net"
import "time"

type SocketClient struct {
	PackageChan chan Package
	con         *net.TCPConn
}

func (c *SocketClient) writeRaw(data []byte) (int, error) {
	c.con.SetWriteDeadline(time.Now().Add(20 * time.Second))
	return c.con.Write(data)
}

func (c *SocketClient) sendPack(data []byte) (int, error) {
	return c.writeRaw(data)
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

func (c *SocketClient) Close() error {
	return c.con.Close()
}

func MakeSocketClient(con *net.TCPConn) SocketClient {
	con.SetReadDeadline(time.Now().Add(20 * time.Second))
	client := SocketClient{make(chan Package), con}
	goingClose := make(chan bool, 1)
	reader := NewSocketReader()

	go func() {
		for {
			select {
			case <-goingClose:
				// the connection is going to close
				return
			default:
				var buffer []byte = make([]byte, 128)
				outBytes, err := con.Read(buffer)
				con.SetReadDeadline(time.Now().Add(10 * time.Second))
				if err != nil {
					con.Close()
				}
				if outBytes == 0 {
					time.Sleep(1 * time.Second)
				} else {
					reader.OnData(buffer)
				}
			}

		}
	}()
	go func() {
		for {
			select {
			case <-goingClose:
				// the connection is going to close
				return
			case pkg := <-reader.PackageChan:
				// pipe Package into public scope
				client.PackageChan <- pkg
			}
		}
	}()
	return client
}
