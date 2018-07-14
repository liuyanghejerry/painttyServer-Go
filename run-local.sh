#!/bin/sh

export GOPATH=`pwd`
go run -race src/server/painttyServer.go