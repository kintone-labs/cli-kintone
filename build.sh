#!/bin/bash

GOOS=linux GOARCH=amd64 go build -o build/linux-x64/cli-kintone
GOOS=darwin GOARCH=amd64 go build -o build/macos-x64/cli-kintone
GOOS=windows GOARCH=amd64 go build -o build/windows-x64/cli-kintone.exe

zip -r cli-kintone.zip build

