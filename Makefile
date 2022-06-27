TAG=$(shell git describe --tags --abbrev=10 --dirty --long)
DIR ?= $(CURDIR)

.PHONY: dev
dev:
	@echo " + $@"
	@go install github.com/golang/protobuf/protoc-gen-go
	# @go install github.com/gogo/protobuf/protoc-gen-gogofast
	@wget -q -O protoc.zip "https://github.com/google/protobuf/releases/download/v3.14.0/protoc-3.14.0-linux-x86_64.zip"
	@unzip -d protoc-tmp protoc.zip
	@go install github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint
	@go install github.com/favadi/protoc-go-inject-tag

style:
	@goimports -w .

test:
	@echo " + $@"

generated-srcs: proto-generated-srcs
	@echo " + $@"

proto-generated-srcs:
	@echo " + $@"
	@scripts/generate-proto-srcs.sh

build: generated-srcs
	@echo " + $@"
	@scripts/go-build.sh

image: build
	@echo " + $@"
	@docker build -t wolfinger/varangian:$(TAG) image/
