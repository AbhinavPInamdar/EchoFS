#!/bin/bash

echo "Installing protoc tools..."
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

echo "Generating gRPC code..."
protoc --go_out=. --go-grpc_out=. proto/v1/echofs.proto

echo "gRPC code generated successfully!"