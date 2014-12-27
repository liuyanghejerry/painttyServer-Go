package main

import (
	"RoomManager"
	"Router"
	"Socket"
	"encoding/json"
	"flag"
	"log"
	"net"
	"os"
	"syscall"
	"time"
)

var painttyServer = ``

var args = []string{
	"painttyServer",
}

var env = []string{
	"a=0",
}

var workingDir = ``

func init() {
	flag.StringVar(&workingDir, "wd", ".", "working path of painttyServer")
	flag.StringVar(&painttyServer, "server", "./painttyServer", "path of painttyServer")
	flag.Parse()
}

func startProc() *os.Process {
	proc, err := os.StartProcess(painttyServer,
		args,
		&os.ProcAttr{
			Dir: workingDir,
			//Env:   env,
			Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
			Sys:   &syscall.SysProcAttr{},
		})
	if err != nil {
		log.Println(err)
		panic(err)
	}
	return proc
}

func dial() *Socket.SocketClient {
	tcpAddr, err := net.ResolveTCPAddr("tcp", "localhost:7777")
	if err != nil {
		panic(nil)
	}
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		panic(err)
	}
	return Socket.MakeSocketClient(conn)
}

func sendMessage(client *Socket.SocketClient) {
	raw, err := json.Marshal(RoomManager.RoomListRequest{
		Request: "roomlist",
	})
	if err != nil {
		panic(err)
	}
	client.SendManagerPack(raw)
}

func loop(client *Socket.SocketClient) <-chan bool {
	router := Router.MakeRouter("response")
	seemsDead := make(chan bool)

	go func(client *Socket.SocketClient) {
		for {
			select {
			case <-time.After(time.Second * 10):
				sendMessage(client)
			case _, _ = <-client.GoingClose:
				return
			}
		}
	}(client)

	go func(client *Socket.SocketClient, dead chan<- bool) {
		for {
			select {
			case pkg, ok := <-client.PackageChan:
				if !ok {
					return
				}
				if pkg.PackageType == Socket.MANAGER {
					err := router.OnMessage(pkg.Unpacked, client)
					if err != nil {
						client.Close()
					}
				}
			case <-time.After(time.Second * 30):
				dead <- true
			case _, _ = <-client.GoingClose:
				return
			}
		}
	}(client, seemsDead)

	return seemsDead
}

func watch(proc *os.Process, ch <-chan bool) {
	procChan := make(chan bool)
	go func(ch chan bool) {
		proc.Wait()
		ch <- true
	}(procChan)
	select {
	case <-ch:
		proc.Kill()
		proc.Release()
	case <-procChan:
	}
}

func main() {
	for {
		proc := startProc()
		<-time.After(time.Second * 10)
		client := dial()
		seemsDead := loop(client)
		watch(proc, seemsDead)
	}
	return
}
