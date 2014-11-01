package Socket

import "Common"

type Package struct {
	PackageType int
	Unpacked    []byte
	Repacked    []byte
}

type SocketReader struct {
	buffer      []byte
	dataSize    int
	PackageChan chan Package
}

func NewSocketReader() SocketReader {
	reader := SocketReader{
		buffer:      make([]byte, 0),
		dataSize:    0,
		PackageChan: make(chan Package),
	}
	return reader
}

func (r *SocketReader) OnData(chunk []byte) (err error) {
	err = nil
	r.buffer = append(r.buffer, chunk...)

	var GET_PACKAGE_SIZE_FROM_DATA = func() int {
		var pg_size_array = r.buffer[0:4]
		r.buffer = r.buffer[4:]
		var pg_size = int(pg_size_array[0])<<24 + int(pg_size_array[1])<<16 + int(pg_size_array[2])<<8 + int(pg_size_array[3])
		return pg_size
	}

	var READ_RAW_BYTES = func(size int) []byte {
		var data = r.buffer[0:size]
		r.buffer = r.buffer[size:]
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
			if len(r.buffer) < 4 {
				break
			}
			r.dataSize = GET_PACKAGE_SIZE_FROM_DATA()
		}
		if len(r.buffer) < r.dataSize {
			break
		}
		var packageData = READ_RAW_BYTES(r.dataSize) // raw single package
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
			func() {
				defer func() { recover() }()
				r.PackageChan <- p
			}()
		} else {
			var p = Package{
				p_header.PackType,
				dataBlock,
				repacked,
			}
			func() {
				defer func() { recover() }()
				r.PackageChan <- p
			}()
		}

		r.dataSize = 0
	}
	return err
}
