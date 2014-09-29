package Config

import (
	"crypto/sha512"
	//"fmt"
	yaml "gopkg.in/yaml.v1"
	"os"
	//"path/filepath"
	"github.com/dustin/randbo"
)

var confMap map[interface{}]interface{}

func init() {
	confMap = readConfMap("./config.yml")
	saltFileName, ok := confMap["salt"].(string)
	if !ok {
		saltFileName = "salt.key"
	}

	salt, err := readSaltFromFile(saltFileName)

	if err != nil {
		salt = createSaltFile()
	}

	var saltHash = make([]byte, 64, 64)
	for i, v := range sha512.Sum512(salt) {
		saltHash[i] = v
	}
	confMap["globalSaltHash"] = saltHash
}

func GetConfig() map[interface{}]interface{} {
	return confMap
}

func createSaltFile() []byte {
	file, err := os.Create("./salt.key")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	var buf = make([]byte, 4096)
	randbo.New().Read(buf)

	file.Write(buf)
	return buf
}

func readSaltFromFile(saltFileName string) ([]byte, error) {
	saltFile, err := os.Open(saltFileName)
	if err != nil {
		return make([]byte, 0), err
	}

	defer saltFile.Close()

	fi, err := saltFile.Stat()
	if err != nil {
		return make([]byte, 0), err
	}

	buf := make([]byte, fi.Size())
	_, err = saltFile.Read(buf)
	if err != nil {
		return make([]byte, 0), err
	}

	return buf, nil
}

func readConfMap(confFileName string) (conf map[interface{}]interface{}) {
	defer func() {
		if r := recover(); r != nil {
			conf = make(map[interface{}]interface{})
		}
	}()

	file, err := os.Open(confFileName)
	if _, ok := err.(*os.PathError); ok {
		file, err = os.Create(confFileName)
		if err != nil {
			panic(err)
		}
	} else {
		panic(err)
	}

	defer file.Close()

	fi, err := file.Stat()
	if err != nil {
		panic(err)
	}

	if fi.Size() <= 0 {
		panic("config file is empty")
	}

	var buf = make([]byte, fi.Size())
	file.Read(buf)
	var tmp interface{}
	err = yaml.Unmarshal(buf, &tmp)
	if err != nil {
		panic(err)
	}

	return tmp.(map[interface{}]interface{})
}
