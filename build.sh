#!/bin/bash

BIN_VERSION="xkfastdfs:${1-not set}"

#for linux
GOPATH=`pwd` GOOS=linux GOARCH=amd64 go build -o xfs -ldflags "-w -s -X 'main.VERSION=$BIN_VERSION' -X 'main.BUILD_TIME=build_time:`date`' -X 'main.GO_VERSION=`go version`' -X 'main.GIT_VERSION=git_version:`git rev-parse HEAD`'" xk.go
GOPATH=`pwd` GOOS=linux GOARCH=386 go build -o xfs32 -ldflags "-w -s -X 'main.VERSION=$BIN_VERSION' -X 'main.BUILD_TIME=build_time:`date`' -X 'main.GO_VERSION=`go version`' -X 'main.GIT_VERSION=git_version:`git rev-parse HEAD`'" xk.go

#for windows
GOPATH=`pwd` GOOS=windows GOARCH=amd64 go build -o xfs.exe -ldflags "-w -s -X 'main.VERSION=$BIN_VERSION' -X 'main.BUILD_TIME=build_time:`date`' -X 'main.GO_VERSION=`go version`' -X 'main.GIT_VERSION=git_version:`git rev-parse HEAD`'" xk.go
GOPATH=`pwd` GOOS=windows GOARCH=386 go build -o xkfastdfs32.exe -ldflags "-w -s -X 'main.VERSION=$BIN_VERSION' -X 'main.BUILD_TIME=build_time:`date`' -X 'main.GO_VERSION=`go version`' -X 'main.GIT_VERSION=git_version:`git rev-parse HEAD`'" xk.go

#for mac
GOPATH=`pwd` GOOS=darwin GOARCH=amd64 go build -o xfs_darwin -ldflags "-w -s -X 'main.VERSION=$BIN_VERSION' -X 'main.BUILD_TIME=build_time:`date`' -X 'main.GO_VERSION=`go version`' -X 'main.GIT_VERSION=git_version:`git rev-parse HEAD`'" xk.go
GOPATH=`pwd` GOOS=darwin GOARCH=386 go build -o xfs_darwin32 -ldflags "-w -s -X 'main.VERSION=$BIN_VERSION' -X 'main.BUILD_TIME=build_time:`date`' -X 'main.GO_VERSION=`go version`' -X 'main.GIT_VERSION=git_version:`git rev-parse HEAD`'" xk.go
