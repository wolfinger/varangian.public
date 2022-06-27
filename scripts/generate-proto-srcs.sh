#!/bin/bash

set -ex

gateway_path=$(go list -f '{{.Dir}}' -m github.com/grpc-ecosystem/grpc-gateway)
#gogo_path=$(go list -f '{{.Dir}}' -m github.com/gogo/protobuf)
protoc="protoc-tmp/bin/protoc"
#protoc_args=(-Iproto "-I${gogo_path}" "-I${gateway_path}/third_party/googleapis" -Iprotoc-tmp/include)
protoc_args=(-Iproto "-I${gateway_path}/third_party/googleapis" -Iprotoc-tmp/include)

args=()
for file in $(find proto/storage/*.proto); do
  file=${file#proto/}
  args+=("M${file}=github.com/wolfinger/varangian/generated/storage")
done

function join { local IFS=","; echo "$*"; }
margs="$(join ${args[@]})"

mkdir -p generated
protoc-tmp/bin/protoc "${protoc_args[@]}" --proto_path=proto --go_out=${margs},plugins=grpc:generated proto/storage/*.proto
protoc-tmp/bin/protoc "${protoc_args[@]}" --grpc-gateway_out=${margs}:generated --grpc-gateway_opt=logtostderr=true proto/api/v1/*.proto
protoc-tmp/bin/protoc "${protoc_args[@]}" --proto_path=proto --go_out=${margs},plugins=grpc:generated proto/api/v1/*.proto

for file in $(find generated/storage/*.pb.go); do
  protoc-go-inject-tag -input=${file}
done