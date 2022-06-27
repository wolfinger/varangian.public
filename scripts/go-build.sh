#!/bin/bash

set -e

var_stamps=("-X github.com/wolfinger/varangian/pkg/version.Version=123")
ldflags=(-s -w "${var_stamps[@]}") # statically compile the binary and stamp with version

mkdir -p image/bin
GOOS=linux GOARCH=amd64 go build -ldflags="${ldflags[*]}" -o image/bin/varangian cmd/varangian.go