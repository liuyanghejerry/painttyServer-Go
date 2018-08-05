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

var confMap sync.Map
var confMutex = &sync.Mutex{}

func InitConf() {
	workingDir, err := os.Getwd()
	log.Println("workingDir:", workingDir)
	// currentDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	// log.Println("currentDir:", currentDir)
	loadConfFile(filepath.Join(workingDir, "config.yml"))
	applyDefaultString(&confMap, "salt", "./data/salt.key")
	stub, _ := confMap.Load("salt")
	saltFileName, _ := stub.(string)

	salt, err := readSaltFromFile(saltFileName)

	if err != nil {
		salt = createSaltFile()
		log.Println("Cannot read salt file, creating new salt...")
	}

	var saltHash = make([]byte, 64, 64)
	for i, v := range sha512.Sum512(salt) {
		saltHash[i] = v
	}
	confMap.Store("globalSaltHash", saltHash)
	applyDefaultSimpleConfigValues(&confMap)
}

func applyDefaultString(confs *sync.Map, key, defaultValue string) {
	stub, ok := confs.Load(key)
	if !ok {
		log.Printf("%s not set. Using default value (%s)...\n", key, defaultValue)
		confs.Store(key, defaultValue)
		return
	}
	_, ok = stub.(string)
	if ok {
		return
	}
	log.Printf("%s read failed. Using default value (%s)...\n", key, defaultValue)
	confs.Store(key, defaultValue)
}

func applyDefaultInt(confs *sync.Map, key string, defaultValue int) {
	stub, ok := confs.Load(key)
	if !ok {
		log.Printf("%s not set. Using default value (%d)...\n", key, defaultValue)
		confs.Store(key, defaultValue)
		return
	}
	_, ok = stub.(int)
	if ok {
		return
	}
	log.Printf("%s read failed. Using default value (%d)...\n", key, defaultValue)
	confs.Store(key, defaultValue)
}

func applyDefaultSimpleConfigValues(confs *sync.Map) {
	applyDefaultString(confs, "announcement", "")
	applyDefaultInt(confs, "expiration", 48)
	applyDefaultInt(confs, "max_load", 8)
	applyDefaultInt(confs, "max_room_count", 1000)
}

func createSaltFile() []byte {
	workingDir, err := os.Getwd()
	if err != nil {
		log.Println("Cannot create salt file.")
		panic(err)
	}
	file, err := os.Create(filepath.Join(workingDir, "salt.key"))
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

func loadConfFile(confFileName string) {
	file, err := os.Open(confFileName)
	if _, ok := err.(*os.PathError); ok {
		log.Println("Cannot find config file.")
		file, err = os.Create(confFileName)
		if err != nil {
			log.Println("Cannot create config file.")
			panic(err)
		}
	} else if err != nil {
		log.Panicln("Cannot open config file:", err)
	}

	defer file.Close()

	fi, err := file.Stat()
	if err != nil {
		log.Panicln("Cannot read config file info.", err)
	}

	if fi.Size() <= 0 {
		log.Panicln("Config file is empty")
	}

	var buf = make([]byte, fi.Size())
	file.Read(buf)
	var tmp interface{}
	err = yaml.Unmarshal(buf, &tmp)
	if err != nil {
		log.Println("Cannot parse config file(as yaml).")
		panic(err)
	}

	tmpConf := tmp.(map[interface{}]interface{})
	for key, value := range tmpConf {
		confMap.Store(key, value)
	}
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
	loadConfFile(filepath.Join(currentDir, "config.yml"))
	applyDefaultSimpleConfigValues(&confMap)
	log.Println("config reloaded")
}

func ReadConfString(key, defaultValue string) string {
	stub, ok := confMap.Load(key)
	if !ok {
		log.Printf("Using default value for %s\n", key)
		return defaultValue
	}
	value, ok := stub.(string)
	if !ok {
		log.Printf("Using default value for %s\n", key)
		return defaultValue
	}
	return value
}

func ReadConfInt(key string, defaultValue int) int {
	stub, ok := confMap.Load(key)
	if !ok {
		log.Printf("Using default value for %s", key)
		return defaultValue
	}
	value, ok := stub.(int)
	if !ok {
		log.Printf("Using default value for %s", key)
		return defaultValue
	}
	return value
}

func ReadConfBytes(key string) []byte {
	stub, _ := confMap.Load(key)
	value, _ := stub.([]byte)
	return value
}
