package Config

import (
	"crypto/sha512"
	"fmt"
	"github.com/dustin/randbo"
	yaml "gopkg.in/yaml.v1"
	"os"
)

var confMap map[interface{}]interface{}

func init() {
	confMap = readConfMap("./config.yml")
	saltFileName, ok := confMap["salt"].(string)
	if !ok {
		fmt.Println("Salt file cannot be determined by config. Using default salt.key...")
		saltFileName = "salt.key"
	}

	salt, err := readSaltFromFile(saltFileName)

	if err != nil {
		salt = createSaltFile()
		fmt.Println("Cannot read salt file, creating new salt...")
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
		fmt.Println("Cannot create salt file.")
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
		fmt.Println("Cannot open salt file.")
		return make([]byte, 0), err
	}

	defer saltFile.Close()

	fi, err := saltFile.Stat()
	if err != nil {
		fmt.Println("Cannot read salt file info.")
		return make([]byte, 0), err
	}

	buf := make([]byte, fi.Size())
	_, err = saltFile.Read(buf)
	if err != nil {
		fmt.Println("Cannot read salt file.")
		return make([]byte, 0), err
	}

	return buf, nil
}

func readConfMap(confFileName string) (conf map[interface{}]interface{}) {
	defer func() {
		if r := recover(); r != nil || conf == nil {
			conf = make(map[interface{}]interface{})
			fmt.Println("Making temp config...")
		}
	}()

	file, err := os.Open(confFileName)
	if _, ok := err.(*os.PathError); ok {
		fmt.Println("Cannot find config file.")
		file, err = os.Create(confFileName)
		if err != nil {
			fmt.Println("Cannot create config file.")
			panic(err)
		}
	} else if err != nil {
		fmt.Println("Cannot open config file.")
		panic(err)
	}

	defer file.Close()

	fi, err := file.Stat()
	if err != nil {
		fmt.Println("Cannot read config file info.")
		panic(err)
	}

	if fi.Size() <= 0 {
		panic("Config file is empty")
	}

	var buf = make([]byte, fi.Size())
	file.Read(buf)
	var tmp interface{}
	err = yaml.Unmarshal(buf, &tmp)
	if err != nil {
		fmt.Println("Cannot parse config file(as yaml).")
		panic(err)
	}

	conf = tmp.(map[interface{}]interface{})

	return conf
}
