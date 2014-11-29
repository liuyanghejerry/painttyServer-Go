package Config

import (
	"crypto/sha512"
	"github.com/dustin/randbo"
	yaml "gopkg.in/yaml.v1"
	"log"
	"os"
	"path/filepath"
)

var confMap map[interface{}]interface{}

func InitConf() {
	workingDir, err := os.Getwd()
	log.Println("workingDir:", workingDir)
	currentDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	log.Println("currentDir:", currentDir)
	confMap = readConfMap(filepath.Join(currentDir, "config.yml"))
	saltFileName, ok := confMap["salt"].(string)
	if !ok {
		log.Println("Salt file cannot be determined by config. Using default salt.key...")
		saltFileName = "salt.key"
	}

	salt, err := readSaltFromFile(saltFileName)

	if err != nil {
		salt = createSaltFile()
		log.Println("Cannot read salt file, creating new salt...")
	}

	var saltHash = make([]byte, 64, 64)
	for i, v := range sha512.Sum512(salt) {
		saltHash[i] = v
	}
	confMap["globalSaltHash"] = saltHash

	announcement, ok := confMap["announcement"].(string)
	if !ok {
		log.Println("Announcement not set. Using empty announcement...")
		confMap["announcement"] = ""
	} else {
		confMap["announcement"] = announcement
	}

	expiration, ok := confMap["expiration"].(int)
	if !ok {
		log.Println("Expiration not set. Using 48 as default expiration...")
		confMap["expiration"] = 48
	} else {
		confMap["expiration"] = expiration
	}

	maxLoad, ok := confMap["maxLoad"].(int)
	if !ok {
		log.Println("maxLoad not set. Using 8 as default expiration...")
		confMap["maxLoad"] = 48
	} else {
		confMap["maxLoad"] = maxLoad
	}
}

func GetConfig() map[interface{}]interface{} {
	return confMap
}

func createSaltFile() []byte {
	currentDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	file, err := os.Create(filepath.Join(currentDir, "salt.key"))
	if err != nil {
		log.Println("Cannot create salt file.")
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
		log.Println("Cannot open salt file.")
		return make([]byte, 0), err
	}

	defer saltFile.Close()

	fi, err := saltFile.Stat()
	if err != nil {
		log.Println("Cannot read salt file info.")
		return make([]byte, 0), err
	}

	buf := make([]byte, fi.Size())
	_, err = saltFile.Read(buf)
	if err != nil {
		log.Println("Cannot read salt file.")
		return make([]byte, 0), err
	}

	return buf, nil
}

func readConfMap(confFileName string) (conf map[interface{}]interface{}) {
	defer func() {
		if r := recover(); r != nil || conf == nil {
			conf = make(map[interface{}]interface{})
			log.Println("Making temp config...")
		}
	}()

	file, err := os.Open(confFileName)
	if _, ok := err.(*os.PathError); ok {
		log.Println("Cannot find config file.")
		file, err = os.Create(confFileName)
		if err != nil {
			log.Println("Cannot create config file.")
			panic(err)
		}
	} else if err != nil {
		log.Println("Cannot open config file.")
		panic(err)
	}

	defer file.Close()

	fi, err := file.Stat()
	if err != nil {
		log.Println("Cannot read config file info.")
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
		log.Println("Cannot parse config file(as yaml).")
		panic(err)
	}

	conf = tmp.(map[interface{}]interface{})

	return conf
}
