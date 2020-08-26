#!/bin/bash

proto_name="plugin_v2"
protoc_ver="3.13.0"
protoc_gen_go_ver="1.2.0"

licence='/*
Copyright (c) '"$(date +"%Y")"' SolarWinds Worldwide, LLC

    Licensed under the Apache License, Version 2.0 (the "License");
    you may not use this file except in compliance with the License.
    You may obtain a copy of the License at

      http://www.apache.org/licenses/LICENSE-2.0

    Unless required by applicable law or agreed to in writing, software
    distributed under the License is distributed on an "AS IS" BASIS,
    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
    See the License for the specific language governing permissions and
    limitations under the License.
*/'

linter_ignore='//lint:file-ignore SA1019 Ignore deprecated. Should be fixed by protoc generator.'

if [[ $OS == "Windows_NT" ]]; then
    protoc_os="win64"
    ext=".exe"
else
    protoc_os="linux-x86_64"
    ext=""
fi

echo "Checking for goimports"
if ! which "goimports" &> /dev/null ; then
    echo "Installing goimports"
    go get golang.org/x/tools/cmd/goimports
fi

echo "Checking for protoc"
if ! which "protoc${ext}" &> /dev/null ; then
    protoc_path="https://github.com/protocolbuffers/protobuf/releases/download/v${protoc_ver}/protoc-${protoc_ver}-${protoc_os}.zip"
    
    echo "Installing protoc from ${protoc_path}"
    curl -o protoc.zip -L $protoc_path -s
    unzip -p protoc.zip "bin/protoc${ext}" > "protoc${ext}"
    mv "protoc${ext}" $GOPATH/bin
    rm -f protoc.zip
fi

echo "Checking for protoc-gen-go"
if ! which "protoc-gen-go${ext}" &> /dev/null ; then
    echo "Installing protoc-gen-go"
    GO111MODULE=on go get "github.com/golang/protobuf/protoc-gen-go@v${protoc_gen_go_ver}"
fi

echo "Checking for protoc-gen-grpchan"
if ! which "protoc-gen-grpchan${ext}" &> /dev/null ; then
    echo "Installing protoc-gen-grpchan"
    go get github.com/librato/grpchan/cmd/protoc-gen-grpchan
fi

echo "Generating pb.go files"
protoc --go_out=plugins=grpc:. --grpchan_out=. "${proto_name}".proto

echo "Applying licence and modifications"
echo -e "${licence}\n${linter_ignore}\n"  | cat - "${proto_name}.pb.go" > temp && mv temp "${proto_name}.pb.go"
echo -e "${licence}\n" | cat - "${proto_name}.pb.grpchan.go" > temp && mv temp "${proto_name}.pb.grpchan.go"
goimports -w "${proto_name}.pb.go"
goimports -w "${proto_name}.pb.grpchan.go"
