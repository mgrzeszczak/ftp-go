#!/bin/bash
export GOPATH=`pwd`
cd src/github.com/mgrzeszczak/go-ftp
echo "Formatting source files..."
go fmt
echo "Running tests..."
go test
echo "Compiling..."
go install
