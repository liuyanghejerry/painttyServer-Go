package Config

import (
	"crypto/sha512"
	"fmt"
	yaml "gopkg.in/yaml.v1"
	"os"
	//"path/filepath"
)

var confMap map[interface{}]interface{}

func init() {
	file, err := os.Open("./config.yml")
	if err != nil {
		panic(err)
	}

	defer file.Close()

	fi, err := file.Stat()
	if err != nil {
		panic(err)
	}

	var buf = make([]byte, fi.Size())
	file.Read(buf)
	var tmp interface{}
	err = yaml.Unmarshal(buf, &tmp)
	if err != nil {
		panic(err)
	}

	confMap = tmp.(map[interface{}]interface{})
	fmt.Println(confMap)
	saltFileName := confMap["salt"].(string)
	saltFile, err := os.Open(saltFileName)
	if err != nil {
		panic(err)
	}
	fi, err = saltFile.Stat()
	if err != nil {
		panic(err)
	}
	buf = make([]byte, fi.Size())
	_, err = saltFile.Read(buf)
	if err != nil {
		panic(err)
	}
	var dbuf = make([]byte, 64, 64)
	for i, v := range sha512.Sum512(buf) {
		dbuf[i] = v
	}
	confMap["globalSaltHash"] = dbuf
}

func GetConfig() map[interface{}]interface{} {
	return confMap
}
