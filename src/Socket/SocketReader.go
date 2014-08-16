package Socket

import "bytes"
import "Common"

type Package struct {
	PackageType int
	Unpacked    []byte
	Repacked    []byte
}

type SocketReader struct {
	buffer      bytes.Buffer
	dataSize    int
	PackageChan chan Package
}

func NewSocketReader() SocketReader {
	return SocketReader{
		*new(bytes.Buffer),
		0,
		make(chan Package),
	}
}

func (r *SocketReader) OnData(chunk []byte) {
	r.buffer.Write(chunk)
	var buf = r.buffer

	var GET_PACKAGE_SIZE_FROM_DATA = func() int {
		var pg_size_array []byte = []byte{0, 0, 0, 0}
		buf.Read(pg_size_array)
		var pg_size = int(pg_size_array[0]<<24) + int(pg_size_array[1]<<16) + int(pg_size_array[2]<<8) + int(pg_size_array[3])
		return pg_size
	}

	var READ_RAW_BYTES = func(size int) []byte {
		var data []byte = make([]byte, size)
		buf.Read(data)
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
		if r.dataSize == 0 {
			if buf.Len() < 4 {
				break
			}
			r.dataSize = GET_PACKAGE_SIZE_FROM_DATA()
		}
		if buf.Len() < r.dataSize {
			break
		}

		var packageData = READ_RAW_BYTES(r.dataSize) // raw single package
		var p_header = GET_FLAG(packageData)         // 8bits header
		var dataBlock = packageData[1:]              // dataBlock has no header
		var repacked = REBUILD(packageData)          // repacked, should be equal with packageData

		if p_header.Compress {
			var uncompressed_data, err = Common.QUncompress(dataBlock)
			if err == nil {
				var p = Package{
					p_header.PackType,
					uncompressed_data,
					repacked,
				}
				r.PackageChan <- p
			}
		} else {
			var p = Package{
				p_header.PackType,
				dataBlock,
				repacked,
			}
			r.PackageChan <- p
		}

		r.dataSize = 0
	}
}
