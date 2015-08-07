package Config

import (
	"crypto/sha512"
	"github.com/dustin/randbo"
	yaml "gopkg.in/yaml.v1"
	"log"
	"os"
	"path/filepath"
	"sync"
)

var confMap map[interface{}]interface{}
var confMutex = &sync.Mutex{}

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

	// loading reloadable ones
	realoadbles := initReloadableConfs(GetConfig())

	for k, v := range realoadbles {
		confMap[k] = v
	}
}

func initReloadableConfs(confs map[interface{}]interface{}) map[interface{}]interface{} {
	announcement, ok := confs["announcement"].(string)
	if !ok {
		log.Println("Announcement not set. Using empty announcement...")
		confs["announcement"] = ""
	} else {
		confs["announcement"] = announcement
	}

	expiration, ok := confs["expiration"].(int)
	if !ok {
		log.Println("Expiration not set. Using 48 as default expiration...")
		confs["expiration"] = 48
	} else {
		confs["expiration"] = expiration
	}

	maxLoad, ok := confs["max_load"].(int)
	if !ok {
		log.Println("max_load not set. Using 8 as default max_load...")
		confs["max_load"] = 8
	} else {
		confs["max_load"] = maxLoad
	}

	maxRoomCount, ok := confs["max_room_count"].(int)
	if !ok {
		log.Println("max_room_count not set. Using 1000 as default max_room_count...")
		confs["max_room_count"] = 1000
	} else {
		confs["max_room_count"] = maxRoomCount
	}

	return confs
}

func GetConfig() (newConfMap map[interface{}]interface{}) {
	confMutex.Lock()
	defer func() {
		confMutex.Unlock()
	}()

	for k, v := range confMap {
		newConfMap[k] = v
	}

	return newConfMap
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

func ReloadConf() {
	log.Println("reloading config...")
	workingDir, err := os.Getwd()
	if err != nil {
		log.Println("reloading config failed")
		return
	}
	log.Println("workingDir:", workingDir)
	currentDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Println("reloading config failed")
		return
	}
	log.Println("currentDir:", currentDir)
	newConfMap := readConfMap(filepath.Join(currentDir, "config.yml"))
	// load old ones
	oldConfMap := GetConfig()

	// loading reloadable ones
	realoadbles := initReloadableConfs(newConfMap)

	// merge
	for k, v := range realoadbles {
		oldConfMap[k] = v
	}

	// replace
	confMutex.Lock()
	confMap = oldConfMap
	confMutex.Unlock()
	log.Println("config reloaded")
}
