#!/bin/bash
export GOPATH=`pwd`
echo "Formatting source files..."
go fmt
echo "Running tests..."
go test
echo "Compiling..."
go build

if [ ! -d bin ]; then
      mkdir -p bin;
fi

mv ftp-go bin/
