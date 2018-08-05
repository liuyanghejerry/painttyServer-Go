#!/bin/sh

GOPATH=`pwd` go build -o ./bin/painttyServer ./src/server/painttyServer.go
GOPATH=`pwd` go build -o ./bin/watchDog ./src/watchDog/watchDog.go
