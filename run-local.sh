#!/bin/sh

export GOPATH=`pwd`
DEBUG=* go run -race src/server/painttyServer.go