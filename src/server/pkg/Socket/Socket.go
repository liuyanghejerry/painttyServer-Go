package Socket

import (
	// "encoding/hex"
	// xxhash "github.com/cespare/xxhash"
	// "github.com/dustin/randbo"
	"log"
	"net"
	"server/pkg/Common"
	"server/pkg/Hub"
	// "strconv"
	"sync"
	"time"
)

type SocketCloseCallback func()

type Package struct {
	PackageType int
	Unpacked    []byte
	Repacked    []byte
}

type SocketClient struct {
	Hub.Hub
	readLock  sync.Mutex
	writeLock sync.Mutex
	con       *net.TCPConn
	closeFlag sync.Once
	buffer    []byte
	dataSize  int
}

func (c *SocketClient) WriteRaw(data []byte) (int, error) {
	c.writeLock.Lock()
	defer c.writeLock.Unlock()
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
        c.Pub("close", 0)
		c.Clear()
	})
}

func (c *SocketClient) RunReadLoop() {
	for {
		buffer := make([]byte, 128)
		c.readLock.Lock()
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
		err = c.OnData(buffer[0:outBytes])
		if err != nil {
			log.Println(err)
			c.Close()
			break
		}
	}
}

func (c *SocketClient) OnData(chunk []byte) (err error) {
	err = nil
	c.buffer = append(c.buffer, chunk...)

	var GET_PACKAGE_SIZE_FROM_DATA = func() int {
		var pg_size_array = c.buffer[0:4]
		c.buffer = c.buffer[4:]
		var pg_size = int(pg_size_array[0])<<24 + int(pg_size_array[1])<<16 + int(pg_size_array[2])<<8 + int(pg_size_array[3])
		return pg_size
	}

	var READ_RAW_BYTES = func(size int) []byte {
		var data = c.buffer[0:size]
		c.buffer = c.buffer[size:]
		return data
	}

	var REBUILD = func(rawData []byte) []byte {
		return protocolPack(rawData)
	}

	var GET_FLAG = func(pkgData []byte) PackHeader {
		var header PackHeader
		header.Compress = (pkgData[0] & 0x1) == 0x1
		header.PackType = int((pkgData[0] >> 0x1) & MASK)
		return header
	}

	for {
		if c.dataSize == 0 {
			if len(c.buffer) < 4 {
				break
			}
			c.dataSize = GET_PACKAGE_SIZE_FROM_DATA()
		}
		if len(c.buffer) < c.dataSize {
			break
		}
		var packageData = READ_RAW_BYTES(c.dataSize) // raw single package
		var p_header = GET_FLAG(packageData)         // 8bits header
		var dataBlock = packageData[1:]              // dataBlock has no header
		var repacked = REBUILD(packageData)          // repacked, should be equal with packageData
		if p_header.Compress {
			uncompressed_data, err := Common.QUncompress(dataBlock)
			if err != nil {
				return err
			}
			var p = Package{
				p_header.PackType,
				uncompressed_data,
				repacked,
			}
			c.Pub("package", p)
		} else {
			var p = Package{
				p_header.PackType,
				dataBlock,
				repacked,
			}
			c.Pub("package", p)
		}

		c.dataSize = 0
	}
	return err
}

func MakeSocketClient(con *net.TCPConn) *SocketClient {
	client := SocketClient{
        Hub: Hub.MakeHub(),
		con:       con,
		closeFlag: sync.Once{},
		buffer:    make([]byte, 0),
		dataSize:  0,
	}

	con.SetKeepAlive(true)
	con.SetNoDelay(true)
	con.SetLinger(10)

	go client.RunReadLoop()
	return &client
}
