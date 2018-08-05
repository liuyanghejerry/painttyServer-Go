#!/bin/sh

export GOPATH=`pwd`
go build -o ./bin/painttyServer ./src/server/painttyServer.go
go build -o ./bin/watchDog ./src/watchDog/watchDog.go
