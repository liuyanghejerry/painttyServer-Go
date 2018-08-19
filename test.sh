#!/bin/sh

export GOPATH=`pwd`
DEBUG=* go test -race ./...
